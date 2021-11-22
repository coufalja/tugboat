// Copyright 2017-2019 Lei Ni (nilei81@gmail.com) and other contributors.
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

package tugboat

import (
	"fmt"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/coufalja/tugboat/config"
	"github.com/coufalja/tugboat/raftio"
	"github.com/coufalja/tugboat/server"
	"github.com/lni/goutils/logutil"
)

// ErrUnknownTarget is the error returned when the target address of the node
// is unknown.
var ErrUnknownTarget = errors.New("target address unknown")

// Registry is used to manage all known node addresses in the multi raft system.
// The transport layer uses this address registry to locate nodes.
type Registry struct {
	partitioner server.IPartitioner
	validate    config.TargetValidator
	Addr        sync.Map // map of raftio.NodeInfo => string
}

// NewNodeRegistry returns a new Registry object.
func NewNodeRegistry(streamConnections uint64, v config.TargetValidator) *Registry {
	n := &Registry{validate: v}
	if streamConnections > 1 {
		n.partitioner = server.NewFixedPartitioner(streamConnections)
	}
	return n
}

// Close closes the node registry.
func (n *Registry) Close() error { return nil }

// Add adds the specified node and its target info to the registry.
func (n *Registry) Add(clusterID uint64, nodeID uint64, target string) {
	if n.validate != nil && !n.validate(target) {
		plog.Panicf("invalid target %s", target)
	}
	key := raftio.GetNodeInfo(clusterID, nodeID)
	v, ok := n.Addr.LoadOrStore(key, target)
	if ok {
		if v.(string) != target {
			plog.Panicf("inconsistent target for %s, %s:%s",
				logutil.DescribeNode(clusterID, nodeID), v, target)
		}
	}
}

func (n *Registry) getConnectionKey(addr string, clusterID uint64) string {
	if n.partitioner == nil {
		return addr
	}
	return fmt.Sprintf("%s-%d", addr, n.partitioner.GetPartitionID(clusterID))
}

// Remove removes a remote from the node registry.
func (n *Registry) Remove(clusterID uint64, nodeID uint64) {
	n.Addr.Delete(raftio.GetNodeInfo(clusterID, nodeID))
}

// RemoveCluster removes all nodes info associated with the specified cluster.
func (n *Registry) RemoveCluster(clusterID uint64) {
	var toRemove []raftio.NodeInfo
	n.Addr.Range(func(k, v interface{}) bool {
		ni := k.(raftio.NodeInfo)
		if ni.ClusterID == clusterID {
			toRemove = append(toRemove, ni)
		}
		return true
	})
	for _, v := range toRemove {
		n.Addr.Delete(v)
	}
}

// Resolve looks up the Addr of the specified node.
func (n *Registry) Resolve(clusterID uint64, nodeID uint64) (string, string, error) {
	key := raftio.GetNodeInfo(clusterID, nodeID)
	addr, ok := n.Addr.Load(key)
	if !ok {
		return "", "", ErrUnknownTarget
	}
	return addr.(string), n.getConnectionKey(addr.(string), clusterID), nil
}
