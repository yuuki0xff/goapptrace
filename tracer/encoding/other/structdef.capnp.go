package goserbench

// AUTO GENERATED - DO NOT EDIT

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	C "github.com/glycerine/go-capnproto"
)

type CapnpA C.Struct

func NewCapnpA(s *C.Segment) CapnpA       { return CapnpA(s.NewStruct(40, 1)) }
func NewRootCapnpA(s *C.Segment) CapnpA   { return CapnpA(s.NewRootStruct(40, 1)) }
func AutoNewCapnpA(s *C.Segment) CapnpA   { return CapnpA(s.NewStructAR(40, 1)) }
func ReadRootCapnpA(s *C.Segment) CapnpA  { return CapnpA(s.Root(0).ToStruct()) }
func (s CapnpA) Id() int64                { return int64(C.Struct(s).Get64(0)) }
func (s CapnpA) SetId(v int64)            { C.Struct(s).Set64(0, uint64(v)) }
func (s CapnpA) Tag() uint8               { return C.Struct(s).Get8(8) }
func (s CapnpA) SetTag(v uint8)           { C.Struct(s).Set8(8, v) }
func (s CapnpA) Timestamp() int64         { return int64(C.Struct(s).Get64(16)) }
func (s CapnpA) SetTimestamp(v int64)     { C.Struct(s).Set64(16, uint64(v)) }
func (s CapnpA) Frames() C.UInt64List     { return C.UInt64List(C.Struct(s).GetObject(0)) }
func (s CapnpA) SetFrames(v C.UInt64List) { C.Struct(s).SetObject(0, C.Object(v)) }
func (s CapnpA) Gid() int64               { return int64(C.Struct(s).Get64(24)) }
func (s CapnpA) SetGid(v int64)           { C.Struct(s).Set64(24, uint64(v)) }
func (s CapnpA) Txid() uint64             { return C.Struct(s).Get64(32) }
func (s CapnpA) SetTxid(v uint64)         { C.Struct(s).Set64(32, v) }
func (s CapnpA) WriteJSON(w io.Writer) error {
	b := bufio.NewWriter(w)
	var err error
	var buf []byte
	_ = buf
	err = b.WriteByte('{')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"id\":")
	if err != nil {
		return err
	}
	{
		s := s.Id()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(',')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"tag\":")
	if err != nil {
		return err
	}
	{
		s := s.Tag()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(',')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"timestamp\":")
	if err != nil {
		return err
	}
	{
		s := s.Timestamp()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(',')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"frames\":")
	if err != nil {
		return err
	}
	{
		s := s.Frames()
		{
			err = b.WriteByte('[')
			if err != nil {
				return err
			}
			for i, s := range s.ToArray() {
				if i != 0 {
					_, err = b.WriteString(", ")
				}
				if err != nil {
					return err
				}
				buf, err = json.Marshal(s)
				if err != nil {
					return err
				}
				_, err = b.Write(buf)
				if err != nil {
					return err
				}
			}
			err = b.WriteByte(']')
		}
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(',')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"gid\":")
	if err != nil {
		return err
	}
	{
		s := s.Gid()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(',')
	if err != nil {
		return err
	}
	_, err = b.WriteString("\"txid\":")
	if err != nil {
		return err
	}
	{
		s := s.Txid()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte('}')
	if err != nil {
		return err
	}
	err = b.Flush()
	return err
}
func (s CapnpA) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	err := s.WriteJSON(&b)
	return b.Bytes(), err
}
func (s CapnpA) WriteCapLit(w io.Writer) error {
	b := bufio.NewWriter(w)
	var err error
	var buf []byte
	_ = buf
	err = b.WriteByte('(')
	if err != nil {
		return err
	}
	_, err = b.WriteString("id = ")
	if err != nil {
		return err
	}
	{
		s := s.Id()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err = b.WriteString(", ")
	if err != nil {
		return err
	}
	_, err = b.WriteString("tag = ")
	if err != nil {
		return err
	}
	{
		s := s.Tag()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err = b.WriteString(", ")
	if err != nil {
		return err
	}
	_, err = b.WriteString("timestamp = ")
	if err != nil {
		return err
	}
	{
		s := s.Timestamp()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err = b.WriteString(", ")
	if err != nil {
		return err
	}
	_, err = b.WriteString("frames = ")
	if err != nil {
		return err
	}
	{
		s := s.Frames()
		{
			err = b.WriteByte('[')
			if err != nil {
				return err
			}
			for i, s := range s.ToArray() {
				if i != 0 {
					_, err = b.WriteString(", ")
				}
				if err != nil {
					return err
				}
				buf, err = json.Marshal(s)
				if err != nil {
					return err
				}
				_, err = b.Write(buf)
				if err != nil {
					return err
				}
			}
			err = b.WriteByte(']')
		}
		if err != nil {
			return err
		}
	}
	_, err = b.WriteString(", ")
	if err != nil {
		return err
	}
	_, err = b.WriteString("gid = ")
	if err != nil {
		return err
	}
	{
		s := s.Gid()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err = b.WriteString(", ")
	if err != nil {
		return err
	}
	_, err = b.WriteString("txid = ")
	if err != nil {
		return err
	}
	{
		s := s.Txid()
		buf, err = json.Marshal(s)
		if err != nil {
			return err
		}
		_, err = b.Write(buf)
		if err != nil {
			return err
		}
	}
	err = b.WriteByte(')')
	if err != nil {
		return err
	}
	err = b.Flush()
	return err
}
func (s CapnpA) MarshalCapLit() ([]byte, error) {
	b := bytes.Buffer{}
	err := s.WriteCapLit(&b)
	return b.Bytes(), err
}

type CapnpA_List C.PointerList

func NewCapnpAList(s *C.Segment, sz int) CapnpA_List {
	return CapnpA_List(s.NewCompositeList(40, 1, sz))
}
func (s CapnpA_List) Len() int        { return C.PointerList(s).Len() }
func (s CapnpA_List) At(i int) CapnpA { return CapnpA(C.PointerList(s).At(i).ToStruct()) }
func (s CapnpA_List) ToArray() []CapnpA {
	n := s.Len()
	a := make([]CapnpA, n)
	for i := 0; i < n; i++ {
		a[i] = s.At(i)
	}
	return a
}
func (s CapnpA_List) Set(i int, item CapnpA) { C.PointerList(s).Set(i, C.Object(item)) }
