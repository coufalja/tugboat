// Copyright 2017-2020 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package config contains functions and types used for managing dragonboat's
configurations.
*/
package config

import (
	"errors"
	"path/filepath"
	"reflect"

	"github.com/coufalja/tugboat/internal/vfs"
	"github.com/coufalja/tugboat/logger"
	"github.com/coufalja/tugboat/raftio"
	pb "github.com/coufalja/tugboat/raftpb"
	"github.com/lni/goutils/stringutil"
)

var plog = logger.GetLogger("config")

const (
	// don't change these, see the comments on ExpertConfig.
	defaultExecShards  uint64 = 16
	defaultLogDBShards uint64 = 16
)

// CompressionType is the type of the compression.
type CompressionType = pb.CompressionType

const (
	// NoCompression is the CompressionType value used to indicate not to use
	// any compression.
	NoCompression = pb.NoCompression
	// Snappy is the CompressionType value used to indicate that google snappy
	// is used for data compression.
	Snappy = pb.Snappy
)

// Config is used to configure Raft nodes.
type Config struct {
	// NodeID is a non-zero value used to identify a node within a Raft cluster.
	NodeID uint64
	// ClusterID is the unique value used to identify a Raft cluster.
	ClusterID uint64
	// CheckQuorum specifies whether the leader node should periodically check
	// non-leader node status and step down to become a follower node when it no
	// longer has the quorum.
	CheckQuorum bool
	// Whether to use PreVote for this node. PreVote is described in the section
	// 9.7 of the raft thesis.
	PreVote bool
	// ElectionRTT is the minimum number of message RTT between elections. Message
	// RTT is defined by NodeHostConfig.RTTMillisecond. The Raft paper suggests it
	// to be a magnitude greater than HeartbeatRTT, which is the interval between
	// two heartbeats. In Raft, the actual interval between elections is
	// randomized to be between ElectionRTT and 2 * ElectionRTT.
	//
	// As an example, assuming NodeHostConfig.RTTMillisecond is 100 millisecond,
	// to set the election interval to be 1 second, then ElectionRTT should be set
	// to 10.
	//
	// When CheckQuorum is enabled, ElectionRTT also defines the interval for
	// checking leader quorum.
	ElectionRTT uint64
	// HeartbeatRTT is the number of message RTT between heartbeats. Message
	// RTT is defined by NodeHostConfig.RTTMillisecond. The Raft paper suggest the
	// heartbeat interval to be close to the average RTT between nodes.
	//
	// As an example, assuming NodeHostConfig.RTTMillisecond is 100 millisecond,
	// to set the heartbeat interval to be every 200 milliseconds, then
	// HeartbeatRTT should be set to 2.
	HeartbeatRTT uint64
	// SnapshotEntries defines how often the state machine should be snapshotted
	// automcatically. It is defined in terms of the number of applied Raft log
	// entries. SnapshotEntries can be set to 0 to disable such automatic
	// snapshotting.
	//
	// When SnapshotEntries is set to N, it means a snapshot is created for
	// roughly every N applied Raft log entries (proposals). This also implies
	// that sending N log entries to a follower is more expensive than sending a
	// snapshot.
	//
	// Once a snapshot is generated, Raft log entries covered by the new snapshot
	// can be compacted. This involves two steps, redundant log entries are first
	// marked as deleted, then they are physically removed from the underlying
	// storage when a LogDB compaction is issued at a later stage. See the godoc
	// on CompactionOverhead for details on what log entries are actually removed
	// and compacted after generating a snapshot.
	//
	// Once automatic snapshotting is disabled by setting the SnapshotEntries
	// field to 0, users can still use NodeHost's RequestSnapshot or
	// SyncRequestSnapshot methods to manually request snapshots.
	SnapshotEntries uint64
	// CompactionOverhead defines the number of most recent entries to keep after
	// each Raft log compaction. Raft log compaction is performance automatically
	// every time when a snapshot is created.
	//
	// For example, when a snapshot is created at let's say index 10,000, then all
	// Raft log entries with index <= 10,000 can be removed from that node as they
	// have already been covered by the created snapshot image. This frees up the
	// maximum storage space but comes at the cost that the full snapshot will
	// have to be sent to the follower if the follower requires any Raft log entry
	// at index <= 10,000. When CompactionOverhead is set to say 500, Dragonboat
	// then compacts the Raft log up to index 9,500 and keeps Raft log entries
	// between index (9,500, 1,0000]. As a result, the node can still replicate
	// Raft log entries between index (9,500, 1,0000] to other peers and only fall
	// back to stream the full snapshot if any Raft log entry with index <= 9,500
	// is required to be replicated.
	CompactionOverhead uint64
	// OrderedConfigChange determines whether Raft membership change is enforced
	// with ordered config change ID.
	//
	// When set to true, ConfigChangeIndex is required for membership change
	// requests. This behaves like an optimistic write lock forcing clients to
	// linearize membership change requests explicitly. (recommended)
	//
	// When set to false (default), ConfigChangeIndex is ignored for membership
	// change requests. This may cause a client to request a membership change
	// based on stale membership data.
	OrderedConfigChange bool
	// MaxInMemLogSize is the target size in bytes allowed for storing in memory
	// Raft logs on each Raft node. In memory Raft logs are the ones that have
	// not been applied yet.
	// MaxInMemLogSize is a target value implemented to prevent unbounded memory
	// growth, it is not for precisely limiting the exact memory usage.
	// When MaxInMemLogSize is 0, the target is set to math.MaxUint64. When
	// MaxInMemLogSize is set and the target is reached, error will be returned
	// when clients try to make new proposals.
	// MaxInMemLogSize is recommended to be significantly larger than the biggest
	// proposal you are going to use.
	MaxInMemLogSize uint64
	// SnapshotCompressionType is the compression type to use for compressing
	// generated snapshot data. No compression is used by default.
	SnapshotCompressionType CompressionType
	// EntryCompressionType is the compression type to use for compressing the
	// payload of user proposals. When Snappy is used, the maximum proposal
	// payload allowed is roughly limited to 3.42GBytes. No compression is used
	// by default.
	EntryCompressionType CompressionType
	// DisableAutoCompactions disables auto compaction used for reclaiming Raft
	// log entry storage spaces. By default, compaction request is issued every
	// time when a snapshot is created, this helps to reclaim disk spaces as
	// soon as possible at the cost of immediate higher IO overhead. Users can
	// disable such auto compactions and use NodeHost.RequestCompaction to
	// manually request such compactions when necessary.
	DisableAutoCompactions bool
	// IsNonVoting indicates whether this is a non-voting Raft node. Described as
	// non-voting members in the section 4.2.1 of Diego Ongaro's thesis, they are
	// used to allow a new node to join the cluster and catch up with other
	// existing ndoes without impacting the availability. Extra non-voting nodes
	// can also be introduced to serve read-only requests.
	IsNonVoting bool
	// IsObserver indicates whether this is a non-voting Raft node without voting
	// power.
	//
	// Deprecated: use IsNonVoting instead.
	IsObserver bool
	// IsWitness indicates whether this is a witness Raft node without actual log
	// replication and do not have state machine. It is mentioned in the section
	// 11.7.2 of Diego Ongaro's thesis.
	//
	// Witness support is currently experimental.
	IsWitness bool
}

// Validate validates the Config instance and return an error when any member
// field is considered as invalid.
func (c *Config) Validate() error {
	if c.NodeID == 0 {
		return errors.New("invalid NodeID, it must be >= 1")
	}
	if c.HeartbeatRTT == 0 {
		return errors.New("HeartbeatRTT must be > 0")
	}
	if c.ElectionRTT == 0 {
		return errors.New("ElectionRTT must be > 0")
	}
	if c.ElectionRTT <= 2*c.HeartbeatRTT {
		return errors.New("invalid election rtt")
	}
	if c.ElectionRTT < 10*c.HeartbeatRTT {
		plog.Warningf("ElectionRTT is not a magnitude larger than HeartbeatRTT")
	}
	if c.MaxInMemLogSize > 0 &&
		c.MaxInMemLogSize < pb.NonCmdFieldSize+1 {
		return errors.New("MaxInMemLogSize is too small")
	}
	if c.SnapshotCompressionType != Snappy &&
		c.SnapshotCompressionType != NoCompression {
		return errors.New("unknown compression type")
	}
	if c.EntryCompressionType != Snappy &&
		c.EntryCompressionType != NoCompression {
		return errors.New("unknown compression type")
	}
	if c.IsWitness && c.SnapshotEntries > 0 {
		return errors.New("witness node can not take snapshot")
	}
	if c.IsObserver {
		c.IsNonVoting = true
	}
	if c.IsWitness && c.IsNonVoting {
		return errors.New("witness node can not be a non-voting node")
	}
	return nil
}

// NodeHostConfig is the configuration used to configure NodeHost instances.
type NodeHostConfig struct {
	// DeploymentID is used to determine whether two NodeHost instances belong to
	// the same deployment and thus allowed to communicate with each other. This
	// helps to prvent accidentially misconfigured NodeHost instances to cause
	// data corruption errors by sending out of context messages to unrelated
	// Raft nodes.
	// For a particular dragonboat based application, you can set DeploymentID
	// to the same uint64 value on all production NodeHost instances, then use
	// different DeploymentID values on your staging and dev environment. It is
	// also recommended to use different DeploymentID values for different
	// dragonboat based applications.
	// When not set, the default value 0 will be used as the deployment ID and
	// thus allowing all NodeHost instances with deployment ID 0 to communicate
	// with each other.
	DeploymentID uint64
	// WALDir is the directory used for storing the WAL of Raft entries. It is
	// recommended to use low latency storage such as NVME SSD with power loss
	// protection to store such WAL data. Leave WALDir to have zero value will
	// have everything stored in NodeHostDir.
	WALDir string
	// NodeHostDir is where everything else is stored.
	NodeHostDir string
	// RTTMillisecond defines the average Rround Trip Time (RTT) in milliseconds
	// between two NodeHost instances. Such a RTT interval is internally used as
	// a logical clock tick, Raft heartbeat and election intervals are both
	// defined in term of how many such logical clock ticks (RTT intervals).
	// Note that RTTMillisecond is the combined delays between two NodeHost
	// instances including all delays caused by network transmission, delays
	// caused by NodeHost queuing and processing. As an example, when fully
	// loaded, the average Rround Trip Time between two of our NodeHost instances
	// used for benchmarking purposes is up to 500 microseconds when the ping time
	// between them is 100 microseconds. Set RTTMillisecond to 1 when it is less
	// than 1 million in your environment.
	RTTMillisecond uint64
	// RaftAddress is a DNS name:port or IP:port address used by the transport
	// module for exchanging Raft messages, snapshots and metadata between
	// NodeHost instances. It should be set to the public address that can be
	// accessed from remote NodeHost instances.
	//
	// When the NodeHostConfig.ListenAddress field is empty, NodeHost listens on
	// RaftAddress for incoming Raft messages. When hostname or domain name is
	// used, it will be resolved to IPv4 addresses first and Dragonboat listens
	// to all resolved IPv4 addresses.
	//
	// By default, the RaftAddress value is not allowed to change between NodeHost
	// restarts. AddressByNodeHostID should be set to true when the RaftAddress
	// value might change after restart.
	RaftAddress string
	// RaftEventListener is the listener for Raft events, such as Raft leadership
	// change, exposed to user space. NodeHost uses a single dedicated goroutine
	// to invoke all RaftEventListener methods one by one, CPU intensive or IO
	// related procedures that can cause long delays should be offloaded to worker
	// goroutines managed by users. See the raftio.IRaftEventListener definition
	// for more details.
	RaftEventListener raftio.IRaftEventListener
	// SystemEventsListener allows users to be notified for system events such
	// as snapshot creation, log compaction and snapshot streaming. It is usually
	// used for testing purposes or for other advanced usages, Dragonboat
	// applications are not required to explicitly set this field.
	SystemEventListener raftio.ISystemEventListener
	// MaxSendQueueSize is the maximum size in bytes of each send queue.
	// Once the maximum size is reached, further replication messages will be
	// dropped to restrict memory usage. When set to 0, it means the send queue
	// size is unlimited.
	MaxSendQueueSize uint64
	// MaxReceiveQueueSize is the maximum size in bytes of each receive queue.
	// Once the maximum size is reached, further replication messages will be
	// dropped to restrict memory usage. When set to 0, it means the queue size
	// is unlimited.
	MaxReceiveQueueSize uint64
	// NotifyCommit specifies whether clients should be notified when their
	// regular proposals and config change requests are committed. By default,
	// commits are not notified, clients are only notified when their proposals
	// are both committed and applied.
	NotifyCommit bool
	// Expert contains options for expert users who are familiar with the internals
	// of Dragonboat. Users are recommended not to use this field unless
	// absoloutely necessary. It is important to note that any change to this field
	// may cause an existing instance unable to restart, it may also cause negative
	// performance impacts.
	Expert ExpertConfig
}

// IFS is the filesystem interface used by tests.
type IFS = vfs.IFS

// TargetValidator is the validtor used to validate user specified target values.
type TargetValidator func(string) bool

// RaftAddressValidator is the validator used to validate user specified
// RaftAddress values.
type RaftAddressValidator func(string) bool

// TransportFactory is the interface used for creating custom transport modules.
type TransportFactory interface {
	// Create creates a transport module.
	Create(NodeHostConfig,
		raftio.MessageHandler, raftio.ChunkHandler) raftio.ITransport
	// Validate validates the RaftAddress of the NodeHost. When using a custom
	// transport module, users are granted full control on what address type to
	// use for the NodeHostConfig.RaftAddress field, it can be of the traditional
	// IP:Port format or any other form. The Validate method is used to validate
	// that a received address is of the valid form.
	Validate(string) bool
}

// RaftRPCFactoryFunc is the factory function that creates the transport module
// instance for exchanging Raft messages between NodeHosts.
//
// Deprecated: Use TransportFactory instead.
type RaftRPCFactoryFunc func(NodeHostConfig,
	raftio.MessageHandler, raftio.ChunkHandler) raftio.ITransport

// Validate validates the NodeHostConfig instance and return an error when
// the configuration is considered as invalid.
func (c *NodeHostConfig) Validate() error {
	if c.RTTMillisecond == 0 {
		return errors.New("invalid RTTMillisecond")
	}
	if len(c.NodeHostDir) == 0 {
		return errors.New("NodeHostConfig.NodeHostDir is empty")
	}
	if c.MaxSendQueueSize > 0 &&
		c.MaxSendQueueSize < pb.NonCmdFieldSize+1 {
		return errors.New("MaxSendQueueSize value is too small")
	}
	if c.MaxReceiveQueueSize > 0 &&
		c.MaxReceiveQueueSize < pb.NonCmdFieldSize+1 {
		return errors.New("MaxReceiveSize value is too small")
	}
	validate := c.GetRaftAddressValidator()
	if !validate(c.RaftAddress) {
		return errors.New("invalid NodeHost address")
	}
	if !c.Expert.Engine.IsEmpty() {
		if err := c.Expert.Engine.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Prepare sets the default value for NodeHostConfig.
func (c *NodeHostConfig) Prepare() error {
	var err error
	c.NodeHostDir, err = filepath.Abs(c.NodeHostDir)
	if err != nil {
		return err
	}
	if len(c.WALDir) > 0 {
		c.WALDir, err = filepath.Abs(c.WALDir)
		if err != nil {
			return err
		}
	}
	if c.Expert.FS == nil {
		c.Expert.FS = vfs.DefaultFS
	}
	if c.Expert.Engine.IsEmpty() {
		plog.Infof("using default EngineConfig")
		c.Expert.Engine = GetDefaultEngineConfig()
	}
	return nil
}

// GetDeploymentID returns the deployment ID to be used.
func (c *NodeHostConfig) GetDeploymentID() uint64 {
	if c.DeploymentID == 0 {
		return 1
	}
	return c.DeploymentID
}

// GetTargetValidator returns a TargetValidator based on the specified
// NodeHostConfig instance.
func (c *NodeHostConfig) GetTargetValidator() TargetValidator {
	return stringutil.IsValidAddress
}

// GetRaftAddressValidator creates a RaftAddressValidator based on the specified
// NodeHostConfig instance.
func (c *NodeHostConfig) GetRaftAddressValidator() RaftAddressValidator {
	return stringutil.IsValidAddress
}

// IsValidAddress returns a boolean value indicating whether the input address
// is valid.
func IsValidAddress(addr string) bool {
	return stringutil.IsValidAddress(addr)
}

// EngineConfig is the configuration for the execution engine.
type EngineConfig struct {
	// ExecShards is the number of execution shards in the first stage of the
	// execution engine. Default value is 16. Once deployed, this value can not
	// be changed later.
	ExecShards uint64
	// CommitShards is the number of commit shards in the second stage of the
	// execution engine. Default value is 16.
	CommitShards uint64
	// ApplyShards is the number of apply shards in the third stage of the
	// execution engine. Default value is 16.
	ApplyShards uint64
	// SnapshotShards is the number of snapshot shards in the forth stage of the
	// execution engine. Default value is 48.
	SnapshotShards uint64
	// CloseShards is the number of close shards used for closing stopped
	// state machines. Default value is 32.
	CloseShards uint64
}

// GetDefaultEngineConfig returns the default EngineConfig instance.
func GetDefaultEngineConfig() EngineConfig {
	return EngineConfig{
		ExecShards:     defaultExecShards,
		CommitShards:   16,
		ApplyShards:    16,
		SnapshotShards: 48,
		CloseShards:    32,
	}
}

// IsEmpty returns a boolean value indicating whether EngineConfig is an empty
// one.
func (ec EngineConfig) IsEmpty() bool {
	return reflect.DeepEqual(&ec, &EngineConfig{})
}

// Validate return an error value when the EngineConfig is invalid.
func (ec EngineConfig) Validate() error {
	if ec.ExecShards == 0 || ec.CommitShards == 0 || ec.ApplyShards == 0 ||
		ec.SnapshotShards == 0 || ec.CloseShards == 0 {
		return errors.New("invalid engine configuration")
	}
	return nil
}

// GetDefaultExpertConfig returns the default ExpertConfig.
func GetDefaultExpertConfig() ExpertConfig {
	return ExpertConfig{
		Engine: GetDefaultEngineConfig(),
	}
}

// ExpertConfig contains options for expert users who are familiar with the
// internals of Dragonboat. Users are recommended not to set ExpertConfig
// unless it is absoloutely necessary.
type ExpertConfig struct {
	// Engine is the cponfiguration for the execution engine.
	Engine EngineConfig
	// FS is the filesystem instance used in tests.
	FS IFS
	// TestNodeHostID is the NodeHostID value to be used by the NodeHost instance.
	// This field is expected to be used in tests only.
	TestNodeHostID uint64
}
