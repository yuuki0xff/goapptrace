// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: structdef-gogo.proto

/*
	Package goserbench is a generated protocol buffer package.

	It is generated from these files:
		structdef-gogo.proto

	It has these top-level messages:
		GogoProtoBufA
*/
package goserbench

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type GogoProtoBufA struct {
	ID        int64    `protobuf:"varint,1,req,name=ID" json:"ID"`
	Tag       uint64   `protobuf:"varint,2,req,name=Tag" json:"Tag"`
	Timestamp int64    `protobuf:"varint,3,req,name=Timestamp" json:"Timestamp"`
	Frames    []uint64 `protobuf:"varint,4,rep,name=Frames" json:"Frames,omitempty"`
	GID       int64    `protobuf:"varint,5,req,name=GID" json:"GID"`
	TxID      uint64   `protobuf:"varint,6,req,name=TxID" json:"TxID"`
}

func (m *GogoProtoBufA) Reset()                    { *m = GogoProtoBufA{} }
func (m *GogoProtoBufA) String() string            { return proto.CompactTextString(m) }
func (*GogoProtoBufA) ProtoMessage()               {}
func (*GogoProtoBufA) Descriptor() ([]byte, []int) { return fileDescriptorStructdefGogo, []int{0} }

func (m *GogoProtoBufA) GetID() int64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *GogoProtoBufA) GetTag() uint64 {
	if m != nil {
		return m.Tag
	}
	return 0
}

func (m *GogoProtoBufA) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *GogoProtoBufA) GetFrames() []uint64 {
	if m != nil {
		return m.Frames
	}
	return nil
}

func (m *GogoProtoBufA) GetGID() int64 {
	if m != nil {
		return m.GID
	}
	return 0
}

func (m *GogoProtoBufA) GetTxID() uint64 {
	if m != nil {
		return m.TxID
	}
	return 0
}

func init() {
	proto.RegisterType((*GogoProtoBufA)(nil), "goserbench.GogoProtoBufA")
}
func (m *GogoProtoBufA) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GogoProtoBufA) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0x8
	i++
	i = encodeVarintStructdefGogo(dAtA, i, uint64(m.ID))
	dAtA[i] = 0x10
	i++
	i = encodeVarintStructdefGogo(dAtA, i, uint64(m.Tag))
	dAtA[i] = 0x18
	i++
	i = encodeVarintStructdefGogo(dAtA, i, uint64(m.Timestamp))
	if len(m.Frames) > 0 {
		for _, num := range m.Frames {
			dAtA[i] = 0x20
			i++
			i = encodeVarintStructdefGogo(dAtA, i, uint64(num))
		}
	}
	dAtA[i] = 0x28
	i++
	i = encodeVarintStructdefGogo(dAtA, i, uint64(m.GID))
	dAtA[i] = 0x30
	i++
	i = encodeVarintStructdefGogo(dAtA, i, uint64(m.TxID))
	return i, nil
}

func encodeVarintStructdefGogo(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *GogoProtoBufA) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovStructdefGogo(uint64(m.ID))
	n += 1 + sovStructdefGogo(uint64(m.Tag))
	n += 1 + sovStructdefGogo(uint64(m.Timestamp))
	if len(m.Frames) > 0 {
		for _, e := range m.Frames {
			n += 1 + sovStructdefGogo(uint64(e))
		}
	}
	n += 1 + sovStructdefGogo(uint64(m.GID))
	n += 1 + sovStructdefGogo(uint64(m.TxID))
	return n
}

func sovStructdefGogo(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozStructdefGogo(x uint64) (n int) {
	return sovStructdefGogo(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GogoProtoBufA) Unmarshal(dAtA []byte) error {
	var hasFields [1]uint64
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowStructdefGogo
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GogoProtoBufA: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GogoProtoBufA: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			m.ID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ID |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000001)
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tag", wireType)
			}
			m.Tag = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Tag |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000002)
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			m.Timestamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Timestamp |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000004)
		case 4:
			if wireType == 0 {
				var v uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowStructdefGogo
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.Frames = append(m.Frames, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowStructdefGogo
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= (int(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthStructdefGogo
				}
				postIndex := iNdEx + packedLen
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				for iNdEx < postIndex {
					var v uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowStructdefGogo
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= (uint64(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.Frames = append(m.Frames, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field Frames", wireType)
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GID", wireType)
			}
			m.GID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.GID |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000008)
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxID", wireType)
			}
			m.TxID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TxID |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			hasFields[0] |= uint64(0x00000010)
		default:
			iNdEx = preIndex
			skippy, err := skipStructdefGogo(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthStructdefGogo
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	if hasFields[0]&uint64(0x00000001) == 0 {
		return proto.NewRequiredNotSetError("ID")
	}
	if hasFields[0]&uint64(0x00000002) == 0 {
		return proto.NewRequiredNotSetError("Tag")
	}
	if hasFields[0]&uint64(0x00000004) == 0 {
		return proto.NewRequiredNotSetError("Timestamp")
	}
	if hasFields[0]&uint64(0x00000008) == 0 {
		return proto.NewRequiredNotSetError("GID")
	}
	if hasFields[0]&uint64(0x00000010) == 0 {
		return proto.NewRequiredNotSetError("TxID")
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipStructdefGogo(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowStructdefGogo
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowStructdefGogo
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthStructdefGogo
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowStructdefGogo
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipStructdefGogo(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthStructdefGogo = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowStructdefGogo   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("structdef-gogo.proto", fileDescriptorStructdefGogo) }

var fileDescriptorStructdefGogo = []byte{
	// 222 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x29, 0x2e, 0x29, 0x2a,
	0x4d, 0x2e, 0x49, 0x49, 0x4d, 0xd3, 0x4d, 0xcf, 0x4f, 0xcf, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9,
	0x17, 0xe2, 0x4a, 0xcf, 0x2f, 0x4e, 0x2d, 0x4a, 0x4a, 0xcd, 0x4b, 0xce, 0x90, 0xd2, 0x4d, 0xcf,
	0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x07, 0x29, 0xd1, 0x07, 0x2b, 0x49, 0x2a,
	0x4d, 0x03, 0xf3, 0xc0, 0x1c, 0x7d, 0x84, 0x56, 0xa5, 0xd5, 0x8c, 0x5c, 0xbc, 0xee, 0xf9, 0xe9,
	0xf9, 0x01, 0x20, 0x9e, 0x53, 0x69, 0x9a, 0xa3, 0x90, 0x08, 0x17, 0x93, 0xa7, 0x8b, 0x04, 0xa3,
	0x02, 0x93, 0x06, 0xb3, 0x13, 0xcb, 0x89, 0x7b, 0xf2, 0x0c, 0x41, 0x4c, 0x9e, 0x2e, 0x42, 0x62,
	0x5c, 0xcc, 0x21, 0x89, 0xe9, 0x12, 0x4c, 0x0a, 0x4c, 0x1a, 0x2c, 0x50, 0x61, 0x90, 0x80, 0x90,
	0x12, 0x17, 0x67, 0x48, 0x66, 0x6e, 0x6a, 0x71, 0x49, 0x62, 0x6e, 0x81, 0x04, 0x33, 0x92, 0x26,
	0x84, 0xb0, 0x90, 0x18, 0x17, 0x9b, 0x5b, 0x51, 0x62, 0x6e, 0x6a, 0xb1, 0x04, 0x8b, 0x02, 0xb3,
	0x06, 0x4b, 0x10, 0x94, 0x07, 0x32, 0xd3, 0xdd, 0xd3, 0x45, 0x82, 0x15, 0x49, 0x17, 0x48, 0x40,
	0x48, 0x82, 0x8b, 0x25, 0xa4, 0xc2, 0xd3, 0x45, 0x82, 0x0d, 0xc9, 0x32, 0xb0, 0x88, 0x93, 0xc0,
	0x89, 0x47, 0x72, 0x8c, 0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0x38, 0xe1, 0xb1, 0x1c,
	0x03, 0x20, 0x00, 0x00, 0xff, 0xff, 0x7f, 0x13, 0x8f, 0xf7, 0x11, 0x01, 0x00, 0x00,
}
