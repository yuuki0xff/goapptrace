// ************************************************************
// This file is automatically generated by genxdr. Do not edit.
// ************************************************************

package goserbench

import (
	"github.com/calmh/xdr"
)

/*

XDRA Structure:

 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                         ID (64 bits)                          +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                 24 zero bits                  |      Tag      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Timestamp (64 bits)                      +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Number of Frames                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                                                               /
|                                                               |
+                       Frames (64 bits)                        +
|                                                               |
/                                                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                         GID (64 bits)                         +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                        Tx ID (64 bits)                        +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+


struct XDRA {
	hyper ID;
	unsigned int Tag;
	hyper Timestamp;
	unsigned hyper Frames<>;
	hyper GID;
	unsigned hyper TxID;
}

*/

func (o XDRA) XDRSize() int {
	return 8 + 4 + 8 +
		4 + len(o.Frames)*8 + 8 + 8
}

func (o XDRA) MarshalXDR() ([]byte, error) {
	buf := make([]byte, o.XDRSize())
	m := &xdr.Marshaller{Data: buf}
	return buf, o.MarshalXDRInto(m)
}

func (o XDRA) MustMarshalXDR() []byte {
	bs, err := o.MarshalXDR()
	if err != nil {
		panic(err)
	}
	return bs
}

func (o XDRA) MarshalXDRInto(m *xdr.Marshaller) error {
	m.MarshalUint64(uint64(o.ID))
	m.MarshalUint8(o.Tag)
	m.MarshalUint64(uint64(o.Timestamp))
	m.MarshalUint32(uint32(len(o.Frames)))
	for i := range o.Frames {
		m.MarshalUint64(o.Frames[i])
	}
	m.MarshalUint64(uint64(o.GID))
	m.MarshalUint64(o.TxID)
	return m.Error
}

func (o *XDRA) UnmarshalXDR(bs []byte) error {
	u := &xdr.Unmarshaller{Data: bs}
	return o.UnmarshalXDRFrom(u)
}
func (o *XDRA) UnmarshalXDRFrom(u *xdr.Unmarshaller) error {
	o.ID = int64(u.UnmarshalUint64())
	o.Tag = u.UnmarshalUint8()
	o.Timestamp = int64(u.UnmarshalUint64())
	_FramesSize := int(u.UnmarshalUint32())
	if _FramesSize < 0 {
		return xdr.ElementSizeExceeded("Frames", _FramesSize, 0)
	} else if _FramesSize == 0 {
		o.Frames = nil
	} else {
		if _FramesSize <= len(o.Frames) {
			o.Frames = o.Frames[:_FramesSize]
		} else {
			o.Frames = make([]uint64, _FramesSize)
		}
		for i := range o.Frames {
			o.Frames[i] = u.UnmarshalUint64()
		}
	}
	o.GID = int64(u.UnmarshalUint64())
	o.TxID = u.UnmarshalUint64()
	return u.Error
}
