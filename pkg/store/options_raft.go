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
	"errors"
	"flag"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"golang.org/x/exp/slog"

	"github.com/webmeshproj/node/pkg/util"
)

const (
	RaftListenAddressEnvVar   = "RAFT_LISTEN_ADDRESS"
	DataDirEnvVar             = "RAFT_DATA_DIR"
	InMemoryEnvVar            = "RAFT_IN_MEMORY"
	ConnectionPoolCountEnvVar = "RAFT_CONNECTION_POOL_COUNT"
	ConnectionTimeoutEnvVar   = "RAFT_CONNECTION_TIMEOUT"
	HeartbeatTimeoutEnvVar    = "RAFT_HEARTBEAT_TIMEOUT"
	ElectionTimeoutEnvVar     = "RAFT_ELECTION_TIMEOUT"
	ApplyTimeoutEnvVar        = "RAFT_APPLY_TIMEOUT"
	CommitTimeoutEnvVar       = "RAFT_COMMIT_TIMEOUT"
	MaxAppendEntriesEnvVar    = "RAFT_MAX_APPEND_ENTRIES"
	LeaderLeaseTimeoutEnvVar  = "RAFT_LEADER_LEASE_TIMEOUT"
	SnapshotIntervalEnvVar    = "RAFT_SNAPSHOT_INTERVAL"
	SnapshotThresholdEnvVar   = "RAFT_SNAPSHOT_THRESHOLD"
	SnapshotRetentionEnvVar   = "RAFT_SNAPSHOT_RETENTION"
	ObserverChanBufferEnvVar  = "RAFT_OBSERVER_CHAN_BUFFER"
	RaftLogFormatEnvVar       = "RAFT_LOG_FORMAT"
	RaftLogLevelEnvVar        = "RAFT_LOG_LEVEL"
	RaftPreferIPv6EnvVar      = "RAFT_PREFER_IPV6"
	LeaveOnShutdownEnvVar     = "LEAVE_ON_SHUTDOWN"
	StartupTimeoutEnvVar      = "STARTUP_TIMEOUT"
	ShutdownTimeoutEnvVar     = "SHUTDOWN_TIMEOUT"

	// LogFile is the raft log file.
	LogFile = "raft.log"
	// StableStoreFile is the raft stable store file.
	StableStoreFile = "raft-stable-dat"
	// DataFile is the data file.
	DataFile = "webmesh.sqlite"
	// LocalDataFile is the local data file.
	LocalDataFile = "local.sqlite"
)

// RaftLogFormat is the raft log format.
type RaftLogFormat string

const (
	// RaftLogFormatJSON is the JSON raft log format.
	RaftLogFormatJSON RaftLogFormat = "json"
	// RaftLogFormatProtobuf is the protobuf raft log format.
	RaftLogFormatProtobuf RaftLogFormat = "protobuf"
	// RaftLogFormatProtobufSnappy is the protobuf snappy raft log format.
	RaftLogFormatProtobufSnappy RaftLogFormat = "protobuf+snappy"
)

// IsValid returns if the raft log format is valid.
func (r RaftLogFormat) IsValid() bool {
	switch r {
	case RaftLogFormatJSON, RaftLogFormatProtobuf, RaftLogFormatProtobufSnappy:
		return true
	default:
		return false
	}
}

// RaftOptions are the raft options.
type RaftOptions struct {
	// ListenAddress is the address to listen on for raft.
	ListenAddress string `json:"listen-address,omitempty" yaml:"listen-address,omitempty" toml:"listen-address,omitempty"`
	// DataDir is the directory to store data in.
	DataDir string `json:"data-dir,omitempty" yaml:"data-dir,omitempty" toml:"data-dir,omitempty"`
	// InMemory is if the store should be in memory. This should only be used for testing and ephemeral nodes.
	InMemory bool `json:"in-memory,omitempty" yaml:"in-memory,omitempty" toml:"in-memory,omitempty"`
	// ConnectionPoolCount is the number of connections to pool. If 0, no connection pooling is used.
	ConnectionPoolCount int `json:"connection-pool-count,omitempty" yaml:"connection-pool-count,omitempty" toml:"connection-pool-count,omitempty"`
	// ConnectionTimeout is the timeout for connections.
	ConnectionTimeout time.Duration `json:"connection-timeout,omitempty" yaml:"connection-timeout,omitempty" toml:"connection-timeout,omitempty"`
	// HeartbeatTimeout is the timeout for heartbeats.
	HeartbeatTimeout time.Duration `json:"heartbeat-timeout,omitempty" yaml:"heartbeat-timeout,omitempty" toml:"heartbeat-timeout,omitempty"`
	// ElectionTimeout is the timeout for elections.
	ElectionTimeout time.Duration `json:"election-timeout,omitempty" yaml:"election-timeout,omitempty" toml:"election-timeout,omitempty"`
	// ApplyTimeout is the timeout for applying.
	ApplyTimeout time.Duration `json:"apply-timeout,omitempty" yaml:"apply-timeout,omitempty" toml:"apply-timeout,omitempty"`
	// CommitTimeout is the timeout for committing.
	CommitTimeout time.Duration `json:"commit-timeout,omitempty" yaml:"commit-timeout,omitempty" toml:"commit-timeout,omitempty"`
	// MaxAppendEntries is the maximum number of append entries.
	MaxAppendEntries int `json:"max-append-entries,omitempty" yaml:"max-append-entries,omitempty" toml:"max-append-entries,omitempty"`
	// LeaderLeaseTimeout is the timeout for leader leases.
	LeaderLeaseTimeout time.Duration `json:"leader-lease-timeout,omitempty" yaml:"leader-lease-timeout,omitempty" toml:"leader-lease-timeout,omitempty"`
	// SnapshotInterval is the interval to take snapshots.
	SnapshotInterval time.Duration `json:"snapshot-interval,omitempty" yaml:"snapshot-interval,omitempty" toml:"snapshot-interval,omitempty"`
	// SnapshotThreshold is the threshold to take snapshots.
	SnapshotThreshold uint64 `json:"snapshot-threshold,omitempty" yaml:"snapshot-threshold,omitempty" toml:"snapshot-threshold,omitempty"`
	// SnapshotRetention is the number of snapshots to retain.
	SnapshotRetention uint64 `json:"snapshot-retention,omitempty" yaml:"snapshot-retention,omitempty" toml:"snapshot-retention,omitempty"`
	// ObserverChanBuffer is the buffer size for the observer channel.
	ObserverChanBuffer int `json:"observer-chan-buffer,omitempty" yaml:"observer-chan-buffer,omitempty" toml:"observer-chan-buffer,omitempty"`
	// LogLevel is the log level for the raft backend.
	LogLevel string `json:"log-level,omitempty" yaml:"log-level,omitempty" toml:"log-level,omitempty"`
	// PreferIPv6 is the prefer IPv6 flag.
	PreferIPv6 bool `json:"prefer-ipv6,omitempty" yaml:"prefer-ipv6,omitempty" toml:"prefer-ipv6,omitempty"`
	// LeaveOnShutdown is the leave on shutdown flag.
	LeaveOnShutdown bool `json:"leave-on-shutdown,omitempty" yaml:"leave-on-shutdown,omitempty" toml:"leave-on-shutdown,omitempty"`
	// LogFormat is the log format for the raft backend.
	LogFormat string `json:"raft-log-format,omitempty" yaml:"raft-log-format,omitempty" toml:"raft-log-format,omitempty"`
	// StartupTimeout is the timeout for starting up.
	StartupTimeout time.Duration `json:"startup-timeout,omitempty" yaml:"startup-timeout,omitempty" toml:"startup-timeout,omitempty"`
	// ShutdownTimeout is the timeout for shutting down.
	ShutdownTimeout time.Duration `json:"shutdown-timeout,omitempty" yaml:"shutdown-timeout,omitempty" toml:"shutdown-timeout,omitempty"`
}

// NewRaftOptions returns new raft options with the default values.
func NewRaftOptions() *RaftOptions {
	return &RaftOptions{
		ListenAddress:      ":9443",
		DataDir:            "/var/lib/webmesh/store",
		ConnectionTimeout:  time.Second * 3,
		HeartbeatTimeout:   time.Second * 3,
		ElectionTimeout:    time.Second * 3,
		ApplyTimeout:       time.Second * 10,
		CommitTimeout:      time.Second * 15,
		LeaderLeaseTimeout: time.Second * 3,
		SnapshotInterval:   time.Minute * 5,
		SnapshotThreshold:  50,
		MaxAppendEntries:   16,
		SnapshotRetention:  3,
		ObserverChanBuffer: 100,
		LogFormat:          string(RaftLogFormatProtobufSnappy),
		LogLevel:           "info",
		StartupTimeout:     time.Minute,
		ShutdownTimeout:    time.Minute,
	}
}

// BindFlags binds the flags to the options.
func (o *RaftOptions) BindFlags(fl *flag.FlagSet) {
	fl.StringVar(&o.ListenAddress, "raft.listen-address", util.GetEnvDefault(RaftListenAddressEnvVar, ":9443"),
		"Raft listen address.")
	fl.StringVar(&o.DataDir, "raft.data-dir", util.GetEnvDefault(DataDirEnvVar, "/var/lib/webmesh/store"),
		"Store data directory.")
	fl.BoolVar(&o.InMemory, "raft.in-memory", util.GetEnvDefault(InMemoryEnvVar, "false") == "true",
		"Store data in memory. This should only be used for testing and ephemeral nodes.")
	fl.IntVar(&o.ConnectionPoolCount, "raft.connection-pool-count", util.GetEnvIntDefault(ConnectionPoolCountEnvVar, 0),
		"Raft connection pool count.")
	fl.DurationVar(&o.ConnectionTimeout, "raft.connection-timeout", util.GetEnvDurationDefault(ConnectionTimeoutEnvVar, time.Second*3),
		"Raft connection timeout.")
	fl.DurationVar(&o.HeartbeatTimeout, "raft.heartbeat-timeout", util.GetEnvDurationDefault(HeartbeatTimeoutEnvVar, time.Second*3),
		"Raft heartbeat timeout.")
	fl.DurationVar(&o.ElectionTimeout, "raft.election-timeout", util.GetEnvDurationDefault(ElectionTimeoutEnvVar, time.Second*3),
		"Raft election timeout.")
	fl.DurationVar(&o.ApplyTimeout, "raft.apply-timeout", util.GetEnvDurationDefault(ApplyTimeoutEnvVar, time.Second*10),
		"Raft apply timeout.")
	fl.DurationVar(&o.CommitTimeout, "raft.commit-timeout", util.GetEnvDurationDefault(CommitTimeoutEnvVar, time.Second*15),
		"Raft commit timeout.")
	fl.IntVar(&o.MaxAppendEntries, "raft.max-append-entries", util.GetEnvIntDefault(MaxAppendEntriesEnvVar, 16),
		"Raft max append entries.")
	fl.DurationVar(&o.LeaderLeaseTimeout, "raft.leader-lease-timeout", util.GetEnvDurationDefault(LeaderLeaseTimeoutEnvVar, time.Second*3),
		"Raft leader lease timeout.")
	fl.DurationVar(&o.SnapshotInterval, "raft.snapshot-interval", util.GetEnvDurationDefault(SnapshotIntervalEnvVar, time.Minute*5),
		"Raft snapshot interval.")
	fl.Uint64Var(&o.SnapshotThreshold, "raft.snapshot-threshold", uint64(util.GetEnvIntDefault(SnapshotThresholdEnvVar, 50)),
		"Raft snapshot threshold.")
	fl.Uint64Var(&o.SnapshotRetention, "raft.snapshot-retention", uint64(util.GetEnvIntDefault(SnapshotRetentionEnvVar, 3)),
		"Raft snapshot retention.")
	fl.StringVar(&o.LogLevel, "raft.log-level", util.GetEnvDefault(RaftLogLevelEnvVar, "info"),
		"Raft log level.")
	fl.BoolVar(&o.PreferIPv6, "raft.prefer-ipv6", util.GetEnvDefault(RaftPreferIPv6EnvVar, "false") == "true",
		"Prefer IPv6 when connecting to raft peers.")
	fl.IntVar(&o.ObserverChanBuffer, "raft.observer-chan-buffer", util.GetEnvIntDefault(ObserverChanBufferEnvVar, 100),
		"Raft observer channel buffer size.")
	fl.BoolVar(&o.LeaveOnShutdown, "raft.leave-on-shutdown", util.GetEnvDefault(LeaveOnShutdownEnvVar, "false") == "true",
		"Leave the cluster when the server shuts down.")
	fl.DurationVar(&o.StartupTimeout, "raft.startup-timeout", util.GetEnvDurationDefault(StartupTimeoutEnvVar, time.Minute*3),
		"Timeout for startup.")
	fl.DurationVar(&o.ShutdownTimeout, "raft.shutdown-timeout", util.GetEnvDurationDefault(ShutdownTimeoutEnvVar, time.Minute),
		"Timeout for graceful shutdown.")
	fl.StringVar(&o.LogFormat, "raft.log-format", util.GetEnvDefault(RaftLogFormatEnvVar, string(RaftLogFormatProtobufSnappy)),
		`Raft log format. Valid options are 'json', 'protobuf', and 'protobuf+snappy'.
All nodes must use the same log format for the lifetime of the cluster.`)
}

// Validate validates the raft options.
func (o *RaftOptions) Validate() error {
	if o == nil {
		return errors.New("raft options are nil")
	}
	if o.DataDir == "" && !o.InMemory {
		return errors.New("data directory is required")
	}
	if o.ConnectionPoolCount < 0 {
		return errors.New("connection pool count must be >= 0")
	}
	if o.ConnectionTimeout <= 0 {
		return errors.New("connection timeout must be > 0")
	}
	if o.HeartbeatTimeout <= 0 {
		return errors.New("heartbeat timeout must be > 0")
	}
	if o.ElectionTimeout <= 0 {
		return errors.New("election timeout must be > 0")
	}
	if o.CommitTimeout <= 0 {
		return errors.New("commit timeout must be > 0")
	}
	if o.MaxAppendEntries <= 0 {
		return errors.New("max append entries must be > 0")
	}
	if o.LeaderLeaseTimeout <= 0 {
		return errors.New("leader lease timeout must be > 0")
	}
	if o.SnapshotInterval <= 0 {
		return errors.New("snapshot interval must be > 0")
	}
	if !RaftLogFormat(o.LogFormat).IsValid() {
		return errors.New("invalid raft log format")
	}
	return nil
}

// RaftConfig builds a raft config.
func (o *RaftOptions) RaftConfig(nodeID string) *raft.Config {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	if o.HeartbeatTimeout != 0 {
		config.HeartbeatTimeout = o.HeartbeatTimeout
	}
	if o.ElectionTimeout != 0 {
		config.ElectionTimeout = o.ElectionTimeout
	}
	if o.CommitTimeout != 0 {
		config.CommitTimeout = o.CommitTimeout
	}
	if o.MaxAppendEntries != 0 {
		config.MaxAppendEntries = o.MaxAppendEntries
	}
	if o.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = o.LeaderLeaseTimeout
	}
	if o.SnapshotInterval != 0 {
		config.SnapshotInterval = o.SnapshotInterval
	}
	if o.SnapshotThreshold != 0 {
		config.SnapshotThreshold = o.SnapshotThreshold
	}
	config.LogLevel = hclog.LevelFromString(o.LogLevel).String()
	config.Logger = o.Logger("raft")
	return config
}

// Logger returns a new logger.
func (o *RaftOptions) Logger(name string) hclog.Logger {
	return &hclogAdapter{
		Logger: slog.Default().With("component", name),
		level:  o.LogLevel,
	}
}

// LogFilePath returns the log file path.
func (o *RaftOptions) LogFilePath() string {
	return filepath.Join(o.DataDir, LogFile)
}

// StableStoreFilePath returns the stable store file path.
func (o *RaftOptions) StableStoreFilePath() string {
	return filepath.Join(o.DataDir, StableStoreFile)
}

// DataFilePath returns the data file path.
func (o *RaftOptions) DataFilePath() string {
	return filepath.Join(o.DataDir, DataFile)
}

// LocalDataFilePath returns the local file path.
func (o *RaftOptions) LocalDataFilePath() string {
	return filepath.Join(o.DataDir, LocalDataFile)
}