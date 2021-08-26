// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: raft.proto

package raftpb

type MessageBatch struct {
	Requests      []Message
	DeploymentId  uint64
	SourceAddress string
	BinVer        uint32
}

func (m *MessageBatch) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MessageBatch) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Requests) > 0 {
		for _, msg := range m.Requests {
			dAtA[i] = 0xa
			i++
			i = encodeVarintRaft(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	dAtA[i] = 0x10
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.DeploymentId))
	dAtA[i] = 0x1a
	i++
	i = encodeVarintRaft(dAtA, i, uint64(len(m.SourceAddress)))
	i += copy(dAtA[i:], m.SourceAddress)
	dAtA[i] = 0x20
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.BinVer))
	return i, nil
}

func (m *MessageBatch) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Requests) > 0 {
		for _, e := range m.Requests {
			l = e.Size()
			n += 1 + l + sovRaft(uint64(l))
		}
	}
	n += 1 + sovRaft(uint64(m.DeploymentId))
	l = len(m.SourceAddress)
	n += 1 + l + sovRaft(uint64(l))
	n += 1 + sovRaft(uint64(m.BinVer))
	return n
}
