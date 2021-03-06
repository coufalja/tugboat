// Copyright 2017-2021 Lei Ni (nilei81@gmail.com) and other contributors.
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
//
//
// This file contains code derived from CockroachDB. The asyncSendSnapshot
// method, connectAndProcessSnapshot method and the processSnapshotQueue
// method is similar to the one used in CockroachDB.
//
// Copyright 2014 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package transport

import (
	"errors"
	"sync/atomic"

	"github.com/coufalja/tugboat/raftio"
	pb "github.com/coufalja/tugboat/raftpb"
	"github.com/lni/vfs"
)

const (
	SnapshotChunkSize  uint64 = 2 * 1024 * 1024
	MaxConnectionCount uint64 = 64
)

// SendSnapshot asynchronously sends raft snapshot message to its target.
func (t *Transport[T]) SendSnapshot(m pb.Message) bool {
	if !t.sendSnapshot(m) {
		plog.Errorf("failed to SendResult snapshot to %s", dn(m.ClusterId, m.To))
		t.sendSnapshotNotification(m.ClusterId, m.To, true)
		return false
	}
	return true
}

// GetStreamSink returns a connection used for streaming snapshot.
func (t *Transport[T]) GetStreamSink(clusterID uint64, nodeID uint64) pb.IChunkSink {
	s := t.getStreamSink(clusterID, nodeID)
	if s == nil {
		plog.Errorf("failed to Connect to %s", dn(clusterID, nodeID))
		t.sendSnapshotNotification(clusterID, nodeID, true)
	}
	return s
}

func (t *Transport[T]) getStreamSink(clusterID uint64, nodeID uint64) pb.IChunkSink {
	addr, _, err := t.resolver.Resolve(clusterID, nodeID)
	if err != nil {
		return nil
	}
	if !t.GetCircuitBreaker(addr).Ready() {
		plog.Warningf("circuit breaker for %s is not ready", addr)
		return nil
	}
	key := raftio.GetNodeInfo(clusterID, nodeID)
	if job := t.createJob(key, true, 0); job != nil {
		shutdown := func() {
			atomic.AddUint64(&t.jobs, ^uint64(0))
		}
		t.stopper.RunWorker(func() {
			t.processSnapshot(job, addr)
			shutdown()
		})
		return &Sink{J: job}
	}
	return nil
}

func (t *Transport[T]) sendSnapshot(m pb.Message) bool {
	if !t.doSendSnapshot(m) {
		if err := m.Snapshot.Unref(); err != nil {
			panic(err)
		}
		return false
	}
	return true
}

func (t *Transport[T]) doSendSnapshot(m pb.Message) bool {
	toNodeID := m.To
	clusterID := m.ClusterId
	if m.Type != pb.InstallSnapshot {
		panic("not a snapshot message")
	}
	chunks, err := SplitSnapshotMessage(m)
	if err != nil {
		plog.Errorf("failed to get snapshot chunks %+v", err)
		return false
	}
	addr, _, err := t.resolver.Resolve(clusterID, toNodeID)
	if err != nil {
		return false
	}
	if !t.GetCircuitBreaker(addr).Ready() {
		t.metrics.SnapshotCnnectionFailure()
		return false
	}
	key := raftio.GetNodeInfo(clusterID, toNodeID)
	job := t.createJob(key, false, len(chunks))
	if job == nil {
		return false
	}
	shutdown := func() {
		atomic.AddUint64(&t.jobs, ^uint64(0))
		if err := m.Snapshot.Unref(); err != nil {
			panic(err)
		}
	}
	t.stopper.RunWorker(func() {
		t.processSnapshot(job, addr)
		shutdown()
	})
	job.AddSnapshot(chunks)
	return true
}

func (t *Transport[T]) createJob(key raftio.NodeInfo, streaming bool, sz int) *SnapshotJob {
	if v := atomic.AddUint64(&t.jobs, 1); v > MaxConnectionCount {
		r := atomic.AddUint64(&t.jobs, ^uint64(0))
		plog.Warningf("SnapshotJob count is rate limited %d", r)
		return nil
	}
	job := NewJob(t.ctx, key.ClusterID, key.NodeID, t.deploymentID,
		streaming, sz, t.trans, t.stopper.ShouldStop(), t.fs)
	job.postSend = t.postSend
	job.preSend = t.preSend
	return job
}

func (t *Transport[T]) processSnapshot(c *SnapshotJob, addr string) {
	breaker := t.GetCircuitBreaker(addr)
	successes := breaker.Successes()
	consecFailures := breaker.ConsecFailures()
	clusterID := c.clusterID
	nodeID := c.nodeID
	if err := func() error {
		if err := c.Connect(addr); err != nil {
			plog.Warningf("failed to get snapshot Conn to %s", dn(clusterID, nodeID))
			t.sendSnapshotNotification(clusterID, nodeID, true)
			close(c.failed)
			t.metrics.SnapshotCnnectionFailure()
			return err
		}
		defer c.Close()
		breaker.Success()
		if successes == 0 || consecFailures > 0 {
			plog.Debugf("snapshot stream to %s (%s) established",
				dn(clusterID, nodeID), addr)
			t.sysEvents.ConnectionEstablished(addr, true)
		}
		err := c.Process()
		if err != nil {
			plog.Errorf("snapshot chunk processing failed: %v", err)
		}
		t.sendSnapshotNotification(clusterID, nodeID, err != nil)
		return err
	}(); err != nil {
		plog.Warningf("processSnapshot failed: %v", err)
		breaker.Fail()
		t.sysEvents.ConnectionFailed(addr, true)
	}
}

func (t *Transport[T]) sendSnapshotNotification(clusterID uint64,
	nodeID uint64, rejected bool) {
	if rejected {
		t.metrics.SnapshotSendFailure()
	} else {
		t.metrics.SnapshotSendSuccess()
	}
	t.msgHandler.HandleSnapshotStatus(clusterID, nodeID, rejected)
	plog.Debugf("snapshot notification to %s added, reject %t",
		dn(clusterID, nodeID), rejected)
}

func splitBySnapshotFile(msg pb.Message,
	filepath string, filesize uint64, startChunkID uint64,
	sf *pb.SnapshotFile) []pb.Chunk {
	if filesize == 0 {
		panic("empty file")
	}
	results := make([]pb.Chunk, 0)
	chunkCount := (filesize-1)/SnapshotChunkSize + 1
	for i := uint64(0); i < chunkCount; i++ {
		var csz uint64
		if i == chunkCount-1 {
			csz = filesize - (chunkCount-1)*SnapshotChunkSize
		} else {
			csz = SnapshotChunkSize
		}
		c := pb.Chunk{
			BinVer:         raftio.TransportBinVersion,
			ClusterId:      msg.ClusterId,
			NodeId:         msg.To,
			From:           msg.From,
			FileChunkId:    i,
			FileChunkCount: chunkCount,
			ChunkId:        startChunkID + i,
			ChunkSize:      csz,
			Index:          msg.Snapshot.Index,
			Term:           msg.Snapshot.Term,
			OnDiskIndex:    msg.Snapshot.OnDiskIndex,
			Membership:     msg.Snapshot.Membership,
			Filepath:       filepath,
			FileSize:       filesize,
		}
		if sf != nil {
			c.HasFileInfo = true
			c.FileInfo = *sf
		}
		results = append(results, c)
	}
	return results
}

func getChunks(m pb.Message) []pb.Chunk {
	startChunkID := uint64(0)
	results := splitBySnapshotFile(m,
		m.Snapshot.Filepath, m.Snapshot.FileSize, startChunkID, nil)
	startChunkID += uint64(len(results))
	for _, snapshotFile := range m.Snapshot.Files {
		chunks := splitBySnapshotFile(m,
			snapshotFile.Filepath, snapshotFile.FileSize, startChunkID, snapshotFile)
		results = append(results, chunks...)
		startChunkID += uint64(len(chunks))
	}
	for idx := range results {
		results[idx].ChunkCount = uint64(len(results))
	}
	return results
}

func SplitSnapshotMessage(m pb.Message) ([]pb.Chunk, error) {
	if m.Type != pb.InstallSnapshot {
		panic("not a snapshot message")
	}
	return getChunks(m), nil
}

func loadChunkData(chunk pb.Chunk,
	data []byte, fs vfs.FS) (result []byte, err error) {
	f, err := openChunkFileForRead(chunk.Filepath, fs)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = firstError(err, f.close())
	}()
	offset := chunk.FileChunkId * SnapshotChunkSize
	if chunk.ChunkSize != uint64(len(data)) {
		data = make([]byte, chunk.ChunkSize)
	}
	n, err := f.readAt(data, int64(offset))
	if err != nil {
		return nil, err
	}
	if uint64(n) != chunk.ChunkSize {
		return nil, errors.New("failed to read the snapshot chunk")
	}
	return data, nil
}
