package goserbench

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *A) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "ID"
	o = append(o, 0x86, 0xa2, 0x49, 0x44)
	o = msgp.AppendInt64(o, z.ID)
	// string "Tag"
	o = append(o, 0xa3, 0x54, 0x61, 0x67)
	o = msgp.AppendUint8(o, z.Tag)
	// string "Timestamp"
	o = append(o, 0xa9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	o = msgp.AppendInt64(o, z.Timestamp)
	// string "Frames"
	o = append(o, 0xa6, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Frames)))
	for za0001 := range z.Frames {
		o = msgp.AppendUint64(o, z.Frames[za0001])
	}
	// string "GID"
	o = append(o, 0xa3, 0x47, 0x49, 0x44)
	o = msgp.AppendInt64(o, z.GID)
	// string "TxID"
	o = append(o, 0xa4, 0x54, 0x78, 0x49, 0x44)
	o = msgp.AppendUint64(o, z.TxID)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *A) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "ID":
			z.ID, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Tag":
			z.Tag, bts, err = msgp.ReadUint8Bytes(bts)
			if err != nil {
				return
			}
		case "Timestamp":
			z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Frames":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Frames) >= int(zb0002) {
				z.Frames = (z.Frames)[:zb0002]
			} else {
				z.Frames = make([]uint64, zb0002)
			}
			for za0001 := range z.Frames {
				z.Frames[za0001], bts, err = msgp.ReadUint64Bytes(bts)
				if err != nil {
					return
				}
			}
		case "GID":
			z.GID, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "TxID":
			z.TxID, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *A) Msgsize() (s int) {
	s = 1 + 3 + msgp.Int64Size + 4 + msgp.Uint8Size + 10 + msgp.Int64Size + 7 + msgp.ArrayHeaderSize + (len(z.Frames) * (msgp.Uint64Size)) + 4 + msgp.Int64Size + 5 + msgp.Uint64Size
	return
}
