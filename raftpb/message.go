// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: raft.proto

package raftpb

type Message struct {
	Type      MessageType
	To        uint64
	From      uint64
	ClusterId uint64
	Term      uint64
	LogTerm   uint64
	LogIndex  uint64
	Commit    uint64
	Reject    bool
	Hint      uint64
	Entries   []Entry
	Snapshot  Snapshot
	HintHigh  uint64
}

func (m *Message) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0x8
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.Type))
	dAtA[i] = 0x10
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.To))
	dAtA[i] = 0x18
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.From))
	dAtA[i] = 0x20
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.ClusterId))
	dAtA[i] = 0x28
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.Term))
	dAtA[i] = 0x30
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.LogTerm))
	dAtA[i] = 0x38
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.LogIndex))
	dAtA[i] = 0x40
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.Commit))
	dAtA[i] = 0x48
	i++
	if m.Reject {
		dAtA[i] = 1
	} else {
		dAtA[i] = 0
	}
	i++
	dAtA[i] = 0x50
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.Hint))
	if len(m.Entries) > 0 {
		for _, msg := range m.Entries {
			dAtA[i] = 0x5a
			i++
			i = encodeVarintRaft(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	dAtA[i] = 0x62
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.Snapshot.Size()))
	n2, err := m.Snapshot.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	dAtA[i] = 0x68
	i++
	i = encodeVarintRaft(dAtA, i, uint64(m.HintHigh))
	return i, nil
}

func (m *Message) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 1 + sovRaft(uint64(m.Type))
	n += 1 + sovRaft(uint64(m.To))
	n += 1 + sovRaft(uint64(m.From))
	n += 1 + sovRaft(uint64(m.ClusterId))
	n += 1 + sovRaft(uint64(m.Term))
	n += 1 + sovRaft(uint64(m.LogTerm))
	n += 1 + sovRaft(uint64(m.LogIndex))
	n += 1 + sovRaft(uint64(m.Commit))
	n += 2
	n += 1 + sovRaft(uint64(m.Hint))
	if len(m.Entries) > 0 {
		for _, e := range m.Entries {
			l = e.Size()
			n += 1 + l + sovRaft(uint64(l))
		}
	}
	l = m.Snapshot.Size()
	n += 1 + l + sovRaft(uint64(l))
	n += 1 + sovRaft(uint64(m.HintHigh))
	return n
}
