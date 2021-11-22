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

package tugboat

import (
	"sync/atomic"

	"github.com/coufalja/tugboat/raftio"
	"github.com/coufalja/tugboat/server"
)

type raftEventListener struct {
	leaderID  *uint64
	queue     *leaderInfoQueue
	termValue uint64
	nodeID    uint64
	clusterID uint64
}

var _ server.IRaftEventListener = (*raftEventListener)(nil)

func newRaftEventListener(clusterID uint64, nodeID uint64, leaderID *uint64, queue *leaderInfoQueue) *raftEventListener {
	el := &raftEventListener{
		clusterID: clusterID,
		nodeID:    nodeID,
		leaderID:  leaderID,
		queue:     queue,
	}
	return el
}

func (e *raftEventListener) close() {
}

func (e *raftEventListener) LeaderUpdated(info server.LeaderInfo) {
	atomic.StoreUint64(e.leaderID, info.LeaderID)
	atomic.StoreUint64(&e.termValue, info.Term)
	if e.queue != nil {
		ui := raftio.LeaderInfo{
			ClusterID: info.ClusterID,
			NodeID:    info.NodeID,
			Term:      info.Term,
			LeaderID:  info.LeaderID,
		}
		e.queue.addLeaderInfo(ui)
	}
}

func (e *raftEventListener) CampaignLaunched(server.CampaignInfo) {
}

func (e *raftEventListener) CampaignSkipped(server.CampaignInfo) {
}

func (e *raftEventListener) SnapshotRejected(server.SnapshotInfo) {
}

func (e *raftEventListener) ReplicationRejected(server.ReplicationInfo) {
}

func (e *raftEventListener) ProposalDropped(server.ProposalInfo) {
}

func (e *raftEventListener) ReadIndexDropped(server.ReadIndexInfo) {
}

type sysEventListener struct {
	stopc  chan struct{}
	events chan server.SystemEvent
	ul     raftio.ISystemEventListener
}

func newSysEventListener(l raftio.ISystemEventListener,
	stopc chan struct{}) *sysEventListener {
	return &sysEventListener{
		stopc:  stopc,
		events: make(chan server.SystemEvent),
		ul:     l,
	}
}

func (l *sysEventListener) Publish(e server.SystemEvent) {
	if l.ul == nil {
		return
	}
	select {
	case l.events <- e:
	case <-l.stopc:
		return
	}
}

func (l *sysEventListener) handle(e server.SystemEvent) {
	if l.ul == nil {
		return
	}
	switch e.Type {
	case server.NodeHostShuttingDown:
		l.ul.NodeHostShuttingDown()
	case server.NodeReady:
		l.ul.NodeReady(getNodeInfo(e))
	case server.NodeUnloaded:
		l.ul.NodeUnloaded(getNodeInfo(e))
	case server.MembershipChanged:
		l.ul.MembershipChanged(getNodeInfo(e))
	case server.ConnectionEstablished:
		l.ul.ConnectionEstablished(getConnectionInfo(e))
	case server.ConnectionFailed:
		l.ul.ConnectionFailed(getConnectionInfo(e))
	case server.SendSnapshotStarted:
		l.ul.SendSnapshotStarted(getSnapshotInfo(e))
	case server.SendSnapshotCompleted:
		l.ul.SendSnapshotCompleted(getSnapshotInfo(e))
	case server.SendSnapshotAborted:
		l.ul.SendSnapshotAborted(getSnapshotInfo(e))
	case server.SnapshotReceived:
		l.ul.SnapshotReceived(getSnapshotInfo(e))
	case server.SnapshotRecovered:
		l.ul.SnapshotRecovered(getSnapshotInfo(e))
	case server.SnapshotCreated:
		l.ul.SnapshotCreated(getSnapshotInfo(e))
	case server.SnapshotCompacted:
		l.ul.SnapshotCompacted(getSnapshotInfo(e))
	case server.LogCompacted:
		l.ul.LogCompacted(getEntryInfo(e))
	case server.LogDBCompacted:
		l.ul.LogDBCompacted(getEntryInfo(e))
	default:
		panic("unknown event type")
	}
}

func getSnapshotInfo(e server.SystemEvent) raftio.SnapshotInfo {
	return raftio.SnapshotInfo{
		ClusterID: e.ClusterID,
		NodeID:    e.NodeID,
		From:      e.From,
		Index:     e.Index,
	}
}

func getNodeInfo(e server.SystemEvent) raftio.NodeInfo {
	return raftio.NodeInfo{
		ClusterID: e.ClusterID,
		NodeID:    e.NodeID,
	}
}

func getEntryInfo(e server.SystemEvent) raftio.EntryInfo {
	return raftio.EntryInfo{
		ClusterID: e.ClusterID,
		NodeID:    e.NodeID,
		Index:     e.Index,
	}
}

func getConnectionInfo(e server.SystemEvent) raftio.ConnectionInfo {
	return raftio.ConnectionInfo{
		Address:            e.Address,
		SnapshotConnection: e.SnapshotConnection,
	}
}
