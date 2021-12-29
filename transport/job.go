// Copyright 2017-2021 Lei Ni (nilei81@gmaij.com) and other contributors.
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

package transport

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/coufalja/tugboat/raftio"
	pb "github.com/coufalja/tugboat/raftpb"
	"github.com/lni/goutils/logutil"
	"github.com/lni/vfs"
)

const (
	StreamingChanLength = 4
)

var (
	// ErrStopped is the error returned to indicate that the connection has
	// already been stopped.
	ErrStopped = errors.New("connection stopped")
	// ErrStreamSnapshot is the error returned to indicate that snapshot
	// streaming failed.
	ErrStreamSnapshot = errors.New("stream snapshot failed")
)

// Sink is the chunk sink for receiving generated snapshot chunk.
type Sink struct {
	J *SnapshotJob
}

// Receive receives a snapshot chunk.
func (s *Sink) Receive(chunk pb.Chunk) (bool, bool) {
	return s.J.AddChunk(chunk)
}

// Close closes the sink processing.
func (s *Sink) Close() error {
	s.Receive(pb.Chunk{ChunkCount: pb.PoisonChunkCount})
	return nil
}

// ClusterID returns the cluster ID of the source node.
func (s *Sink) ClusterID() uint64 {
	return s.J.clusterID
}

// ToNodeID returns the node ID of the node intended to get and handle the
// received snapshot chunk.
func (s *Sink) ToNodeID() uint64 {
	return s.J.nodeID
}

type SnapshotJob struct {
	Conn         raftio.ISnapshotConnection
	preSend      atomic.Value
	postSend     atomic.Value
	fs           vfs.FS
	ctx          context.Context
	transport    raftio.ITransport
	Stream       chan pb.Chunk
	completed    chan struct{}
	stopc        chan struct{}
	failed       chan struct{}
	deploymentID uint64
	nodeID       uint64
	clusterID    uint64
	streaming    bool
}

func NewJob(ctx context.Context,
	clusterID uint64, nodeID uint64,
	did uint64, streaming bool, sz int, transport raftio.ITransport,
	stopc chan struct{}, fs vfs.FS) *SnapshotJob {
	j := &SnapshotJob{
		clusterID:    clusterID,
		nodeID:       nodeID,
		deploymentID: did,
		streaming:    streaming,
		ctx:          ctx,
		transport:    transport,
		stopc:        stopc,
		failed:       make(chan struct{}),
		completed:    make(chan struct{}),
		fs:           fs,
	}
	var chsz int
	if streaming {
		chsz = StreamingChanLength
	} else {
		chsz = sz
	}
	j.Stream = make(chan pb.Chunk, chsz)
	return j
}

func (j *SnapshotJob) Close() {
	if j.Conn != nil {
		j.Conn.Close()
	}
}

func (j *SnapshotJob) Connect(addr string) error {
	conn, err := j.transport.GetSnapshotConnection(j.ctx, addr)
	if err != nil {
		plog.Errorf("failed to get a SnapshotJob to %s, %v", addr, err)
		return err
	}
	j.Conn = conn
	return nil
}

func (j *SnapshotJob) AddSnapshot(chunks []pb.Chunk) {
	if len(chunks) != cap(j.Stream) {
		plog.Panicf("cap of Stream is %d, want %d", cap(j.Stream), len(chunks))
	}
	for _, chunk := range chunks {
		j.Stream <- chunk
	}
}

func (j *SnapshotJob) AddChunk(chunk pb.Chunk) (bool, bool) {
	if !chunk.IsPoisonChunk() {
		plog.Debugf("%s is sending chunk %d to %s",
			logutil.NodeID(chunk.From), chunk.ChunkId,
			dn(chunk.ClusterId, chunk.NodeId))
	} else {
		plog.Debugf("sending a poison chunk to %s", dn(j.clusterID, j.nodeID))
	}

	select {
	case j.Stream <- chunk:
		return true, false
	case <-j.completed:
		if !chunk.IsPoisonChunk() {
			plog.Panicf("more chunk received for completed SnapshotJob")
		}
		return true, false
	case <-j.failed:
		plog.Warningf("stream snapshot to %s failed", dn(j.clusterID, j.nodeID))
		return false, false
	case <-j.stopc:
		return false, true
	}
}

func (j *SnapshotJob) Process() error {
	if j.Conn == nil {
		panic("nil connection")
	}
	if j.streaming {
		err := j.streamSnapshot()
		if err != nil {
			close(j.failed)
		}
		return err
	}
	return j.sendSnapshot()
}

func (j *SnapshotJob) streamSnapshot() error {
	for {
		select {
		case <-j.stopc:
			plog.Warningf("stream snapshot to %s stopped", dn(j.clusterID, j.nodeID))
			return ErrStopped
		case chunk := <-j.Stream:
			chunk.DeploymentId = j.deploymentID
			if chunk.IsPoisonChunk() {
				return ErrStreamSnapshot
			}
			if err := j.sendChunk(chunk, j.Conn); err != nil {
				plog.Errorf("streaming snapshot chunk to %s failed, %v",
					dn(chunk.ClusterId, chunk.NodeId), err)
				return err
			}
			if chunk.ChunkCount == pb.LastChunkCount {
				plog.Debugf("node %d just sent all chunks to %s",
					chunk.From, dn(chunk.ClusterId, chunk.NodeId))
				close(j.completed)
				return nil
			}
		}
	}
}

func (j *SnapshotJob) sendSnapshot() error {
	chunks := make([]pb.Chunk, 0)
	for {
		select {
		case <-j.stopc:
			return ErrStopped
		case chunk := <-j.Stream:
			if len(chunks) == 0 && chunk.ChunkId != 0 {
				panic("chunk alignment error")
			}
			chunks = append(chunks, chunk)
			if chunk.ChunkId+1 == chunk.ChunkCount {
				return j.sendChunks(chunks)
			}
		}
	}
}

func (j *SnapshotJob) sendChunks(chunks []pb.Chunk) error {
	chunkData := make([]byte, SnapshotChunkSize)
	for _, chunk := range chunks {
		select {
		case <-j.stopc:
			return ErrStopped
		default:
		}
		chunk.DeploymentId = j.deploymentID
		data, err := loadChunkData(chunk, chunkData, j.fs)
		if err != nil {
			panicNow(err)
		}
		chunk.Data = data
		if err := j.sendChunk(chunk, j.Conn); err != nil {
			return err
		}
		if f := j.postSend.Load(); f != nil {
			f.(func(pb.Chunk))(chunk)
		}
	}
	return nil
}

func (j *SnapshotJob) sendChunk(c pb.Chunk,
	conn raftio.ISnapshotConnection) error {
	if f := j.preSend.Load(); f != nil {
		updated, shouldSend := f.(StreamChunkSendFunc)(c)
		if !shouldSend {
			plog.Debugf("chunk to %s skipped", dn(c.ClusterId, c.NodeId))
			return errChunkSendSkipped
		}
		return conn.SendChunk(updated)
	}
	return conn.SendChunk(c)
}
