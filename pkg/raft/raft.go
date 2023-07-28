/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package raft contains Raft consensus for WebMesh.
package raft

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	"golang.org/x/exp/slog"

	"github.com/webmeshproj/webmesh/pkg/meshdb/snapshots"
	"github.com/webmeshproj/webmesh/pkg/storage"
)

var (
	// ErrStarted is returned when the Raft node is already started.
	ErrStarted = errors.New("raft node already started")
	// ErrClosed is returned when the Raft node is already closed.
	ErrClosed = errors.New("raft node is closed")
	// ErrAlreadyBootstrapped is returned when the Raft node is already bootstrapped.
	ErrAlreadyBootstrapped = raft.ErrCantBootstrap
	// ErrNotLeader is returned when the Raft node is not the leader.
	ErrNotLeader = raft.ErrNotLeader
)

// Raft is the interface for Raft consensus and storage.
type Raft interface {
	// Start starts the Raft node.
	Start(ctx context.Context, opts *StartOptions) error
	// Bootstrap attempts to bootstrap the Raft cluster. If the cluster is already
	// bootstrapped, ErrAlreadyBootstrapped is returned. If the cluster is not
	// bootstrapped and bootstrapping succeeds, the optional callback is called
	// with isLeader flag set to true if the node is the leader, and false otherwise.
	// Any error returned by the callback is returned by Bootstrap.
	Bootstrap(ctx context.Context, opts *BootstrapOptions) error
	// Storage returns the storage. This is only valid after Start is called.
	Storage() storage.Storage
	// Raft returns the Raft instance. This is only valid after Start is called.
	Raft() *raft.Raft
	// Configuration returns the current raft configuration.
	Configuration() raft.Configuration
	// LastAppliedIndex returns the last applied index.
	LastAppliedIndex() uint64
	// ListenPort returns the listen port.
	ListenPort() int
	// IsLeader returns true if the Raft node is the leader.
	IsLeader() bool
	// AddNonVoter adds a non-voting node to the cluster with timeout enforced by the context.
	AddNonVoter(ctx context.Context, id string, addr string) error
	// AddVoter adds a voting node to the cluster with timeout enforced by the context.
	AddVoter(ctx context.Context, id string, addr string) error
	// DemoteVoter demotes a voting node to a non-voting node with timeout enforced by the context.
	DemoteVoter(ctx context.Context, id string) error
	// RemoveServer removes a peer from the cluster with timeout enforced by the context.
	RemoveServer(ctx context.Context, id string, wait bool) error
	// Restore restores the Raft node from a snapshot.
	Restore(rdr io.ReadCloser) error
	// Stop stops the Raft node.
	Stop(ctx context.Context) error
}

// StartOptons are options for starting a Raft node.
type StartOptions struct {
	// NodeID is the node ID.
	NodeID string
}

// BootstrapOptions are options for bootstrapping a Raft node.
type BootstrapOptions struct {
	// AdvertiseAddress is the address to advertise to the other
	// bootstrap nodes. Defaults to localhost:listen_port if empty.
	AdvertiseAddress string
	// Servers are the Raft servers to bootstrap with.
	// Keys are the node IDs, and values are the Raft addresses.
	Servers map[string]string
	// OnBootstrapped is called when the cluster is bootstrapped.
	OnBootstrapped func(isLeader bool) error
}

// New returns a new Raft node.
func New(opts *Options) Raft {
	return newRaftNode(opts)
}

// raftNode is a Raft node. It implements the Raft interface.
type raftNode struct {
	opts                        *Options
	nodeID                      raft.ServerID
	raft                        *raft.Raft
	started                     atomic.Bool
	lastAppliedIndex            atomic.Uint64
	currentTerm                 atomic.Uint64
	listenPort                  int
	raftTransport               *raft.NetworkTransport
	raftSnapshots               raft.SnapshotStore
	logDB                       LogStoreCloser
	stableDB                    StableStoreCloser
	dataDB                      storage.Storage
	snapshotter                 snapshots.Snapshotter
	observer                    *raft.Observer
	observerChan                chan raft.Observation
	observerClose, observerDone chan struct{}
	log                         *slog.Logger
	mu                          sync.Mutex
}

// newRaftNode returns a new Raft node.
func newRaftNode(opts *Options) *raftNode {
	log := slog.Default().With(slog.String("component", "raft"))
	if opts.InMemory {
		log = log.With(slog.String("storage", "memory"))
	} else {
		log = log.With(slog.String("storage", opts.DataDir))
	}
	return &raftNode{opts: opts, log: log}
}

// Start starts the Raft node.
func (r *raftNode) Start(ctx context.Context, opts *StartOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started.Load() {
		return ErrStarted
	}
	r.nodeID = raft.ServerID(opts.NodeID)
	// Ensure the data directories exist if not in-memory
	if !r.opts.InMemory {
		for _, dir := range []string{r.opts.StorePath(), r.opts.DataStoragePath()} {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return fmt.Errorf("raft mkdir %q: %w", dir, err)
			}
		}
	}
	// Create the raft network transport
	r.log.Debug("creating raft network transport")
	sl, err := NewStreamLayer(r.opts.ListenAddress)
	if err != nil {
		return fmt.Errorf("new raft stream layer: %w", err)
	}
	r.listenPort = sl.ListenPort()
	r.raftTransport = raft.NewNetworkTransport(sl,
		r.opts.ConnectionPoolCount,
		r.opts.ConnectionTimeout,
		&logWriter{log: r.log},
	)
	// Create the stores
	r.log.Debug("creating raft stores")
	err = r.createDataStores(ctx)
	if err != nil {
		defer r.raftTransport.Close()
		return fmt.Errorf("create data stores: %w", err)
	}
	r.snapshotter = snapshots.New(r.dataDB)
	handleErr := func(cause error) error {
		defer r.raftTransport.Close()
		defer r.closeDataStores(ctx)
		return cause
	}
	// Check for any existing snapshots
	snapshots, err := r.raftSnapshots.List()
	if err != nil {
		return handleErr(fmt.Errorf("list snapshots: %w", err))
	}
	if len(snapshots) > 0 {
		latest := snapshots[0]
		r.log.Info("restoring from snapshot",
			slog.String("id", latest.ID),
			slog.Int("term", int(latest.Term)),
			slog.Int("index", int(latest.Index)))
		meta, reader, err := r.raftSnapshots.Open(latest.ID)
		if err != nil {
			return handleErr(fmt.Errorf("open snapshot: %w", err))
		}
		defer reader.Close()
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)
		// Restore to the database.
		if err = r.snapshotter.Restore(ctx, io.NopCloser(tee)); err != nil {
			return handleErr(fmt.Errorf("restore snapshot: %w", err))
		}
		// Call snapshot restore hooks
		if r.opts.OnSnapshotRestore != nil {
			r.opts.OnSnapshotRestore(ctx, meta, io.NopCloser(&buf))
		}
		r.currentTerm.Store(latest.Term)
		r.lastAppliedIndex.Store(latest.Index)
	}
	// Create the raft instance.
	r.log.Info("starting raft instance", slog.String("listen-addr", string(r.raftTransport.LocalAddr())))
	r.raft, err = raft.NewRaft(
		r.opts.RaftConfig(opts.NodeID),
		r,
		&MonotonicLogStore{r.logDB},
		r.stableDB,
		r.raftSnapshots,
		r.raftTransport)
	if err != nil {
		return handleErr(fmt.Errorf("new raft: %w", err))
	}
	// Register observers.
	r.observerChan = make(chan raft.Observation, r.opts.ObserverChanBuffer)
	r.observer = raft.NewObserver(r.observerChan, false, func(o *raft.Observation) bool {
		return true
	})
	r.raft.RegisterObserver(r.observer)
	r.observerClose, r.observerDone = r.observe()
	// We're done here.
	r.started.Store(true)
	return nil
}

// Bootstrap attempts to bootstrap the Raft cluster.
func (r *raftNode) Bootstrap(ctx context.Context, opts *BootstrapOptions) error {
	r.mu.Lock()
	if !r.started.Load() {
		r.mu.Unlock()
		return ErrClosed
	}
	if opts.AdvertiseAddress == "" {
		opts.AdvertiseAddress = fmt.Sprintf("localhost:%d", r.listenPort)
	}
	addr, err := r.resolveTCPAddr(ctx, opts.AdvertiseAddress, 3)
	if err != nil {
		r.mu.Unlock()
		return fmt.Errorf("resolve advertise address: %w", err)
	}
	cfg := raft.Configuration{
		Servers: []raft.Server{
			{
				Suffrage: raft.Voter,
				ID:       r.nodeID,
				Address:  raft.ServerAddress(addr.String()),
			},
		},
	}
	if len(opts.Servers) > 0 {
		for nodeID, listenAddres := range opts.Servers {
			if nodeID == string(r.nodeID) {
				continue
			}
			addr, err := r.resolveTCPAddr(ctx, listenAddres, 3)
			if err != nil {
				r.mu.Unlock()
				return fmt.Errorf("resolve server address: %w", err)
			}
			cfg.Servers = append(cfg.Servers, raft.Server{
				Suffrage: raft.Voter,
				ID:       raft.ServerID(nodeID),
				Address:  raft.ServerAddress(addr.String()),
			})
		}
	}
	f := r.raft.BootstrapCluster(cfg)
	err = f.Error()
	if err != nil {
		defer r.mu.Unlock()
		if err == raft.ErrCantBootstrap {
			return ErrAlreadyBootstrapped
		}
		return fmt.Errorf("bootstrap cluster: %w", err)
	}
	// Wait for the leader to be elected.
	for {
		select {
		case <-ctx.Done():
			r.mu.Unlock()
			return fmt.Errorf("bootstrap cluster: %w", ctx.Err())
		default:
			addr, id := r.raft.LeaderWithID()
			if addr == "" && id == "" {
				time.Sleep(time.Millisecond * 500)
				continue
			}
			// We need to unlock before return as the OnBootstrapped
			// callback will want to write to storage.
			r.mu.Unlock()
			if opts.OnBootstrapped == nil {
				return nil
			}
			isLeader := id == r.nodeID
			return opts.OnBootstrapped(isLeader)
		}
	}
}

// Raft returns the Raft instance.
func (r *raftNode) Raft() *raft.Raft {
	return r.raft
}

// Configuration returns the current raft configuration.
func (r *raftNode) Configuration() raft.Configuration {
	if r.raft == nil {
		return raft.Configuration{}
	}
	return r.raft.GetConfiguration().Configuration()
}

// ListenPort returns the listen port.
func (r *raftNode) ListenPort() int {
	return r.listenPort
}

// LastAppliedIndex returns the last applied index.
func (r *raftNode) LastAppliedIndex() uint64 {
	return r.lastAppliedIndex.Load()
}

// IsLeader returns true if the Raft node is the leader.
func (r *raftNode) IsLeader() bool {
	return r.raft.State() == raft.Leader
}

// Storage returns the storage.
func (r *raftNode) Storage() storage.Storage {
	return &raftStorage{r.dataDB, r}
}

// Stop stops the Raft node.
func (r *raftNode) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started.Load() {
		return ErrClosed
	}
	r.log.Debug("stopping raft node")
	defer r.log.Debug("raft node stopped")
	defer r.started.Store(false)
	defer r.raftTransport.Close()
	defer r.closeDataStores(ctx)
	if _, ok := ctx.Deadline(); !ok {
		// Use the configured shutdown timeout
		if r.opts.ShutdownTimeout > 0 {
			var cancel context.CancelFunc
			deadline := time.Now().Add(r.opts.ShutdownTimeout)
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
	}
	// If we were not running in memory, force a snapshot.
	if !r.opts.InMemory {
		r.log.Debug("taking raft storage snapshot")
		err := r.raft.Snapshot().Error()
		if err != nil {
			// Make this non-fatal for now
			r.log.Error("failed to take snapshot", slog.String("error", err.Error()))
		}
	}
	if r.raft.State() == raft.Leader {
		r.log.Debug("raft node is current leader")
		// If we are the leader we need to step down first.
		if r.opts.LeaveOnShutdown {
			// If we are leaving on shutdown, we need to remove ourselves from the cluster.
			r.log.Debug("removing self from cluster")
			if err := r.RemoveServer(ctx, string(r.nodeID), true); err != nil && err != ErrNotLeader {
				return fmt.Errorf("remove self: %w", err)
			}
		}
		// Try to step down again for good measure.
		r.log.Debug("stepping down as leader")
		if err := r.raft.LeadershipTransfer().Error(); err != nil && err != ErrNotLeader {
			return fmt.Errorf("stepdown: %w", err)
		}
	}
	r.log.Debug("shutting down raft node")
	err := r.raft.Shutdown().Error()
	if err != nil {
		return fmt.Errorf("raft shutdown: %w", err)
	}
	return nil
}

// resolveTCPAddr resolves a TCP address with retries and context.
func (r *raftNode) resolveTCPAddr(ctx context.Context, lookup string, maxRetries int) (net.Addr, error) {
	var addr net.Addr
	var err error
	var tries int
	for tries < maxRetries {
		addr, err = net.ResolveTCPAddr("tcp", lookup)
		if err != nil {
			tries++
			err = fmt.Errorf("resolve tcp address: %w", err)
			r.log.Error("failed to resolve advertise address", slog.String("error", err.Error()))
			if tries < maxRetries {
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("%w: %w", err, ctx.Err())
				case <-time.After(time.Second * 1):
					continue
				}
			}
		}
		break
	}
	return addr, err
}