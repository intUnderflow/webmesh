/*
Copyright 2023.

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

package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb"
	"golang.org/x/exp/slog"

	"github.com/webmeshproj/node/pkg/meshdb/models"
	"github.com/webmeshproj/node/pkg/meshdb/snapshots"
)

// Open opens the store.
func (s *store) Open() error {
	if s.open.Load() {
		return ErrOpen
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.opts.StartupTimeout)
	defer cancel()
	log := s.log
	handleErr := func(err error) error {
		if s.raftTransport != nil {
			defer s.raftTransport.Close()
		}
		if s.raft != nil {
			if shutdownErr := s.raft.Shutdown().Error(); shutdownErr != nil {
				err = fmt.Errorf("%w: %v", err, shutdownErr)
			}
		}
		if s.logDB != nil {
			if closeErr := s.logDB.Close(); closeErr != nil {
				err = fmt.Errorf("%w: %v", err, closeErr)
			}
		}
		if s.stableDB != nil {
			if closeErr := s.stableDB.Close(); closeErr != nil {
				err = fmt.Errorf("%w: %v", err, closeErr)
			}
		}
		return err
	}
	var err error
	// Register a raft db driver.
	raftDriverName := uuid.NewString()
	sql.Register(raftDriverName, &raftDBDriver{s})
	// If bootstrap and force are set, clear the data directory.
	if s.opts.Bootstrap && s.opts.ForceBootstrap {
		err = os.RemoveAll(s.opts.DataDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove all %q: %w", s.opts.DataDir, err)
		}
	}
	// Ensure the data and snapshots directory exists.
	if !s.opts.InMemory {
		err = os.MkdirAll(s.opts.DataDir, 0755)
		if err != nil {
			return fmt.Errorf("mkdir %q: %w", s.opts.DataDir, err)
		}
		log = log.With(slog.String("data-dir", s.opts.DataDir))
	} else {
		log = log.With(slog.String("data-dir", ":memory:"))
	}
	// Create the raft network transport
	log.Debug("creating raft network transport")
	s.raftTransport = raft.NewNetworkTransport(s.sl,
		s.opts.ConnectionPoolCount,
		s.opts.ConnectionTimeout,
		&logWriter{log: s.log},
	)
	// Create the raft stores.
	log.Debug("creating boltdb stores")
	if s.opts.InMemory {
		s.logDB = newInmemStore()
		s.stableDB = newInmemStore()
		s.raftSnapshots = raft.NewInmemSnapshotStore()
	} else {
		s.logDB, err = boltdb.NewBoltStore(s.opts.LogFilePath())
		if err != nil {
			return handleErr(fmt.Errorf("new bolt store %q: %w", s.opts.LogFilePath(), err))
		}
		s.stableDB, err = boltdb.NewBoltStore(s.opts.StableStoreFilePath())
		if err != nil {
			return handleErr(fmt.Errorf("new bolt store %q: %w", s.opts.StableStoreFilePath(), err))
		}
		s.raftSnapshots, err = raft.NewFileSnapshotStoreWithLogger(
			s.opts.DataDir,
			int(s.opts.SnapshotRetention),
			s.opts.RaftLogger("snapshots"),
		)
	}
	if err != nil {
		return handleErr(fmt.Errorf("new file snapshot store %q: %w", s.opts.DataDir, err))
	}
	// Create the data stores.
	log.Debug("creating data stores")
	var dataPath, localDataPath string
	if s.opts.InMemory {
		dataPath = ":memory:"
		localDataPath = ":memory:"
	} else {
		dataPath = s.opts.DataFilePath()
		localDataPath = s.opts.LocalDataFilePath()
	}
	s.weakData, err = sql.Open("sqlite", dataPath)
	if err != nil {
		return handleErr(fmt.Errorf("open data sqlite %q: %w", s.opts.DataFilePath(), err))
	}
	// Make sure we use case sensitive collation for the data store.
	_, err = s.weakData.Exec("PRAGMA case_sensitive_like = true;")
	if err != nil {
		return handleErr(fmt.Errorf("set case sensitive like: %w", err))
	}
	s.raftData, err = sql.Open(raftDriverName, "")
	if err != nil {
		return handleErr(fmt.Errorf("open raft sqlite: %w", err))
	}
	s.localData, err = sql.Open("sqlite", localDataPath)
	if err != nil {
		return handleErr(fmt.Errorf("open local sqlite %q: %w", s.opts.LocalDataFilePath(), err))
	}
	s.snapshotter = snapshots.New(s.weakData)
	// Create the raft instance.
	log.Info("starting raft instance",
		slog.String("listen-addr", string(s.raftTransport.LocalAddr())),
		slog.String("advertise-addr", s.opts.AdvertiseAddress),
	)
	s.raft, err = raft.NewRaft(
		s.opts.RaftConfig(s.nodeID), s,
		&monotonicLogStore{s.logDB},
		s.stableDB,
		s.raftSnapshots,
		s.raftTransport)
	if err != nil {
		return handleErr(fmt.Errorf("new raft: %w", err))
	}
	// Bootstrap the cluster if needed.
	if s.opts.Bootstrap {
		// Database gets migrated during bootstrap.
		log.Info("bootstrapping cluster")
		if err = s.bootstrap(ctx); err != nil {
			return handleErr(fmt.Errorf("bootstrap: %w", err))
		}
	} else if s.opts.Join != "" {
		log.Debug("migrating raft database")
		if err = models.MigrateRaftDB(s.weakData); err != nil {
			return fmt.Errorf("raft db migrate: %w", err)
		}
		log.Debug("migrating local database")
		if err = models.MigrateLocalDB(s.localData); err != nil {
			return fmt.Errorf("local db migrate: %w", err)
		}
		ctx, cancel := context.WithTimeout(ctx, s.opts.JoinTimeout)
		defer cancel()
		if err = s.join(ctx, s.opts.Join); err != nil {
			return handleErr(fmt.Errorf("join: %w", err))
		}
	} else {
		// We neither had the bootstrap flag nor the join flag set.
		// This means we are a possibly a single node cluster.
		// Recover our previous wireguard configuration and start up.
		log.Debug("migrating raft database")
		if err = models.MigrateRaftDB(s.weakData); err != nil {
			return fmt.Errorf("raft db migrate: %w", err)
		}
		log.Debug("migrating local database")
		if err = models.MigrateLocalDB(s.localData); err != nil {
			return fmt.Errorf("local db migrate: %w", err)
		}
		if err := s.recoverWireguard(ctx); err != nil {
			return fmt.Errorf("recover wireguard: %w", err)
		}
	}
	// Register observers.
	s.observerChan = make(chan raft.Observation, s.opts.ObserverChanBuffer)
	s.observer = raft.NewObserver(s.observerChan, false, func(o *raft.Observation) bool {
		return true
	})
	s.raft.RegisterObserver(s.observer)
	s.observerClose, s.observerDone = s.observe()
	s.open.Store(true)
	return nil
}

type monotonicLogStore struct{ raft.LogStore }

var _ = raft.MonotonicLogStore(&monotonicLogStore{})

func (m *monotonicLogStore) IsMonotonic() bool {
	return true
}

func newInmemStore() *inMemoryCloser {
	return &inMemoryCloser{raft.NewInmemStore()}
}

type inMemoryCloser struct {
	*raft.InmemStore
}

func (i *inMemoryCloser) Close() error {
	return nil
}
