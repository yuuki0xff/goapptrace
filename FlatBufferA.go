// automatically generated by the FlatBuffers compiler, do not modify

package flatbuffersmodels

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type FlatBufferA struct {
	_tab flatbuffers.Table
}

func GetRootAsFlatBufferA(buf []byte, offset flatbuffers.UOffsetT) *FlatBufferA {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &FlatBufferA{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *FlatBufferA) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *FlatBufferA) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *FlatBufferA) ID() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *FlatBufferA) MutateID(n int64) bool {
	return rcv._tab.MutateInt64Slot(4, n)
}

func (rcv *FlatBufferA) Tag() byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetByte(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *FlatBufferA) MutateTag(n byte) bool {
	return rcv._tab.MutateByteSlot(6, n)
}

func (rcv *FlatBufferA) Timestamp() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *FlatBufferA) MutateTimestamp(n int64) bool {
	return rcv._tab.MutateInt64Slot(8, n)
}

func (rcv *FlatBufferA) Frames(j int) uint64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetUint64(a + flatbuffers.UOffsetT(j*8))
	}
	return 0
}

func (rcv *FlatBufferA) FramesLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *FlatBufferA) GID() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(12))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *FlatBufferA) MutateGID(n int64) bool {
	return rcv._tab.MutateInt64Slot(12, n)
}

func (rcv *FlatBufferA) TxID() uint64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(14))
	if o != 0 {
		return rcv._tab.GetUint64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *FlatBufferA) MutateTxID(n uint64) bool {
	return rcv._tab.MutateUint64Slot(14, n)
}

func FlatBufferAStart(builder *flatbuffers.Builder) {
	builder.StartObject(6)
}
func FlatBufferAAddID(builder *flatbuffers.Builder, ID int64) {
	builder.PrependInt64Slot(0, ID, 0)
}
func FlatBufferAAddTag(builder *flatbuffers.Builder, Tag byte) {
	builder.PrependByteSlot(1, Tag, 0)
}
func FlatBufferAAddTimestamp(builder *flatbuffers.Builder, Timestamp int64) {
	builder.PrependInt64Slot(2, Timestamp, 0)
}
func FlatBufferAAddFrames(builder *flatbuffers.Builder, Frames flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(Frames), 0)
}
func FlatBufferAStartFramesVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(8, numElems, 8)
}
func FlatBufferAAddGID(builder *flatbuffers.Builder, GID int64) {
	builder.PrependInt64Slot(4, GID, 0)
}
func FlatBufferAAddTxID(builder *flatbuffers.Builder, TxID uint64) {
	builder.PrependUint64Slot(5, TxID, 0)
}
func FlatBufferAEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
