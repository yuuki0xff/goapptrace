package goserbench

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/DeDiS/protobuf"
	"github.com/Sereal/Sereal/Go/sereal"
	"github.com/alecthomas/binary"
	"github.com/davecgh/go-xdr/xdr"
	"github.com/glycerine/go-capnproto"
	"github.com/gogo/protobuf/proto"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/hprose/hprose-go"
	"github.com/tinylib/msgp/msgp"
	"gopkg.in/mgo.v2/bson"
	vmihailenco "gopkg.in/vmihailenco/msgpack.v2"
	"zombiezen.com/go/capnproto2"
)

var (
	validate = os.Getenv("VALIDATE")
)

func randString(l int) string {
	buf := make([]byte, l)
	for i := 0; i < (l+1)/2; i++ {
		buf[i] = byte(rand.Intn(256))
	}
	return fmt.Sprintf("%x", buf)[:l]
}

func generate() []*A {
	a := make([]*A, 0, 1000)
	for i := 0; i < 1000; i++ {
		a = append(a, &A{
			ID:        int64(rand.Intn(1000000)),
			Tag:       uint8(rand.Intn(256)),
			Timestamp: int64(rand.Intn(1000000)),
			Frames: []uint64{
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
			},
			GID:  int64(rand.Intn(1000000)),
			TxID: uint64(rand.Intn(1000000)),
		})
	}
	return a
}

func compareUint64(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type Serializer interface {
	Marshal(o interface{}) []byte
	Unmarshal(d []byte, o interface{}) error
	String() string
}

func benchMarshal(b *testing.B, s Serializer) {
	b.StopTimer()
	data := generate()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Marshal(data[rand.Intn(len(data))])
	}
}

func cmpTags(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func cmpAliases(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

func benchUnmarshal(b *testing.B, s Serializer) {
	b.StopTimer()
	data := generate()
	ser := make([][]byte, len(data))
	for i, d := range data {
		o := s.Marshal(d)
		t := make([]byte, len(o))
		copy(t, o)
		ser[i] = t
	}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n := rand.Intn(len(ser))
		o := &A{}
		err := s.Unmarshal(ser[n], o)
		if err != nil {
			b.Fatalf("%s failed to unmarshal: %s (%s)", s, err, ser[n])
		}
		// Validate unmarshalled data.
		if validate != "" {
			i := data[n]
			correct := i == o
			if !correct {
				b.Fatalf("unmarshaled object differed:\n%v\n%v", i, o)
			}
		}
	}
}

func TestMessage(t *testing.T) {
	println(`
A test suite for benchmarking various Go serialization methods.

See README.md for details on running the benchmarks.
`)

}

// github.com/tinylib/msgp

type MsgpSerializer struct{}

func (m MsgpSerializer) Marshal(o interface{}) []byte {
	out, _ := o.(msgp.Marshaler).MarshalMsg(nil)
	return out
}

func (m MsgpSerializer) Unmarshal(d []byte, o interface{}) error {
	_, err := o.(msgp.Unmarshaler).UnmarshalMsg(d)
	return err
}

func (m MsgpSerializer) String() string { return "Msgp" }

func BenchmarkMsgpMarshal(b *testing.B) {
	benchMarshal(b, MsgpSerializer{})
}

func BenchmarkMsgpUnmarshal(b *testing.B) {
	benchUnmarshal(b, MsgpSerializer{})
}

// gopkg.in/vmihailenco/msgpack.v2

type VmihailencoMsgpackSerializer struct{}

func (m VmihailencoMsgpackSerializer) Marshal(o interface{}) []byte {
	d, _ := vmihailenco.Marshal(o)
	return d
}

func (m VmihailencoMsgpackSerializer) Unmarshal(d []byte, o interface{}) error {
	return vmihailenco.Unmarshal(d, o)
}

func (m VmihailencoMsgpackSerializer) String() string {
	return "vmihailenco-msgpack"
}

func BenchmarkVmihailencoMsgpackMarshal(b *testing.B) {
	benchMarshal(b, VmihailencoMsgpackSerializer{})
}

func BenchmarkVmihailencoMsgpackUnmarshal(b *testing.B) {
	benchUnmarshal(b, VmihailencoMsgpackSerializer{})
}

// encoding/json

type JsonSerializer struct{}

func (j JsonSerializer) Marshal(o interface{}) []byte {
	d, _ := json.Marshal(o)
	return d
}

func (j JsonSerializer) Unmarshal(d []byte, o interface{}) error {
	return json.Unmarshal(d, o)
}

func (j JsonSerializer) String() string {
	return "json"
}

func BenchmarkJsonMarshal(b *testing.B) {
	benchMarshal(b, JsonSerializer{})
}

func BenchmarkJsonUnmarshal(b *testing.B) {
	benchUnmarshal(b, JsonSerializer{})
}

// github.com/mailru/easyjson

type EasyJSONSerializer struct{}

func (m EasyJSONSerializer) Marshal(o interface{}) []byte {
	out, _ := o.(*A).MarshalJSON()
	return out
}

func (m EasyJSONSerializer) Unmarshal(d []byte, o interface{}) error {
	err := o.(*A).UnmarshalJSON(d)
	return err
}

func (m EasyJSONSerializer) String() string { return "EasyJson" }

func BenchmarkEasyJsonMarshal(b *testing.B) {
	benchMarshal(b, EasyJSONSerializer{})
}

func BenchmarkEasyJsonUnmarshal(b *testing.B) {
	benchUnmarshal(b, EasyJSONSerializer{})
}

// gopkg.in/mgo.v2/bson

type BsonSerializer struct{}

func (m BsonSerializer) Marshal(o interface{}) []byte {
	d, _ := bson.Marshal(o)
	return d
}

func (m BsonSerializer) Unmarshal(d []byte, o interface{}) error {
	return bson.Unmarshal(d, o)
}

func (j BsonSerializer) String() string {
	return "bson"
}

func BenchmarkBsonMarshal(b *testing.B) {
	benchMarshal(b, BsonSerializer{})
}

func BenchmarkBsonUnmarshal(b *testing.B) {
	benchUnmarshal(b, BsonSerializer{})
}

// encoding/gob

type GobSerializer struct {
	b   bytes.Buffer
	enc *gob.Encoder
	dec *gob.Decoder
}

func (g *GobSerializer) Marshal(o interface{}) []byte {
	g.b.Reset()
	err := g.enc.Encode(o)
	if err != nil {
		panic(err)
	}
	return g.b.Bytes()
}

func (g *GobSerializer) Unmarshal(d []byte, o interface{}) error {
	g.b.Reset()
	g.b.Write(d)
	err := g.dec.Decode(o)
	return err
}

func (g GobSerializer) String() string {
	return "gob"
}

func NewGobSerializer() *GobSerializer {
	s := &GobSerializer{}
	s.enc = gob.NewEncoder(&s.b)
	s.dec = gob.NewDecoder(&s.b)
	err := s.enc.Encode(A{})
	if err != nil {
		panic(err)
	}
	var a A
	err = s.dec.Decode(&a)
	if err != nil {
		panic(err)
	}
	return s
}

func BenchmarkGobMarshal(b *testing.B) {
	s := NewGobSerializer()
	benchMarshal(b, s)
}

func BenchmarkGobUnmarshal(b *testing.B) {
	s := NewGobSerializer()
	benchUnmarshal(b, s)
}

// github.com/davecgh/go-xdr/xdr

type XdrSerializer struct{}

func (x XdrSerializer) Marshal(o interface{}) []byte {
	d, _ := xdr.Marshal(o)
	return d
}

func (x XdrSerializer) Unmarshal(d []byte, o interface{}) error {
	_, err := xdr.Unmarshal(d, o)
	return err
}

func (x XdrSerializer) String() string {
	return "xdr"
}

func BenchmarkXdrMarshal(b *testing.B) {
	benchMarshal(b, XdrSerializer{})
}

func BenchmarkXdrUnmarshal(b *testing.B) {
	benchUnmarshal(b, XdrSerializer{})
}

// github.com/Sereal/Sereal/Go/sereal

type SerealSerializer struct{}

func (s SerealSerializer) Marshal(o interface{}) []byte {
	d, _ := sereal.Marshal(o)
	return d
}

func (s SerealSerializer) Unmarshal(d []byte, o interface{}) error {
	err := sereal.Unmarshal(d, o)
	return err
}

func (s SerealSerializer) String() string {
	return "sereal"
}

func BenchmarkSerealMarshal(b *testing.B) {
	benchMarshal(b, SerealSerializer{})
}

func BenchmarkSerealUnmarshal(b *testing.B) {
	benchUnmarshal(b, SerealSerializer{})
}

// github.com/alecthomas/binary

type BinarySerializer struct{}

func (b BinarySerializer) Marshal(o interface{}) []byte {
	d, _ := binary.Marshal(o)
	return d
}

func (b BinarySerializer) Unmarshal(d []byte, o interface{}) error {
	return binary.Unmarshal(d, o)
}

func (b BinarySerializer) String() string {
	return "binary"
}

func BenchmarkBinaryMarshal(b *testing.B) {
	benchMarshal(b, BinarySerializer{})
}

func BenchmarkBinaryUnmarshal(b *testing.B) {
	benchUnmarshal(b, BinarySerializer{})
}

// github.com/google/flatbuffers/go

type FlatBufferSerializer struct {
	builder *flatbuffers.Builder
}

func (s *FlatBufferSerializer) Marshal(o interface{}) []byte {
	a := o.(*A)
	builder := s.builder

	builder.Reset()

	FlatBufferAStart(builder)
	FlatBufferAAddID(builder, a.ID)
	FlatBufferAAddTag(builder, a.Tag)
	FlatBufferAAddTimestamp(builder, a.Timestamp)
	offset := FlatBufferAStartFramesVector(builder, len(a.Frames))
	FlatBufferAAddFrames(builder, offset)
	FlatBufferAAddGID(builder, a.GID)
	FlatBufferAAddTxID(builder, a.TxID)
	builder.Finish(FlatBufferAEnd(builder))
	return builder.Bytes[builder.Head():]
}

func (s *FlatBufferSerializer) Unmarshal(d []byte, i interface{}) error {
	a := i.(*A)
	o := FlatBufferA{}
	o.Init(d, flatbuffers.GetUOffsetT(d))
	a.ID = o.ID()
	a.Tag = o.Tag()
	a.Timestamp = o.Timestamp()
	a.Frames = make([]uint64, o.FramesLength())
	for _, i := range a.Frames {
		a.Frames[i] = o.Frames(int(i))
	}
	a.GID = o.GID()
	a.TxID = o.TxID()
	return nil
}

func (s *FlatBufferSerializer) String() string {
	return "FlatBuffer"
}

//func BenchmarkFlatBuffersMarshal(b *testing.B) {
//	benchMarshal(b, &FlatBufferSerializer{flatbuffers.NewBuilder(0)})
//}
//
//func BenchmarkFlatBuffersUnmarshal(b *testing.B) {
//	benchUnmarshal(b, &FlatBufferSerializer{flatbuffers.NewBuilder(0)})
//}

// github.com/glycerine/go-capnproto

type CapNProtoSerializer struct {
	buf []byte
	out *bytes.Buffer
}

func (x *CapNProtoSerializer) Marshal(o interface{}) []byte {
	a := o.(*A)
	s := capn.NewBuffer(x.buf)
	c := NewRootCapnpA(s)
	c.SetId(a.ID)
	c.SetTag(a.Tag)
	c.SetTimestamp(a.Timestamp)
	frames := s.NewUInt64List(len(a.Frames))
	for i, v := range a.Frames {
		frames.Set(i, v)
	}
	c.SetFrames(frames)
	c.SetGid(a.GID)
	c.SetTxid(a.TxID)
	x.out.Reset()
	s.WriteTo(x.out)
	x.buf = []byte(s.Data)[:0]
	return x.out.Bytes()
}

func (x *CapNProtoSerializer) Unmarshal(d []byte, i interface{}) error {
	a := i.(*A)
	s, _, _ := capn.ReadFromMemoryZeroCopy(d)
	o := ReadRootCapnpA(s)
	a.ID = o.Id()
	a.Tag = o.Tag()
	a.Timestamp = o.Timestamp()
	frames := o.Frames()
	for i := 0; i < frames.Len(); i++ {
		a.Frames = append(a.Frames, frames.At(i))
	}
	a.GID = o.Gid()
	a.TxID = o.Txid()
	return nil
}

func (x *CapNProtoSerializer) String() string {
	return "CapNProto"
}

func BenchmarkCapNProtoMarshal(b *testing.B) {
	benchMarshal(b, &CapNProtoSerializer{nil, &bytes.Buffer{}})
}

func BenchmarkCapNProtoUnmarshal(b *testing.B) {
	benchUnmarshal(b, &CapNProtoSerializer{nil, &bytes.Buffer{}})
}

// zombiezen.com/go/capnproto2

type CapNProto2Serializer struct {
	arena capnp.Arena
}

func (x *CapNProto2Serializer) Marshal(o interface{}) []byte {
	a := o.(*A)
	m, s, _ := capnp.NewMessage(x.arena)
	c, _ := NewRootCapnp2A(s)
	c.SetId(a.ID)
	c.SetTag(a.Tag)
	c.SetTimestamp(a.Timestamp)

	frames, _ := capnp.NewUInt64List(s, int32(len(a.Frames)))
	for i, v := range a.Frames {
		frames.Set(i, v)
	}
	c.SetFrames(frames)
	c.SetGid(a.GID)
	c.SetTxid(a.TxID)
	b, _ := m.Marshal()
	return b
}

func (x *CapNProto2Serializer) Unmarshal(d []byte, i interface{}) error {
	a := i.(*A)
	m, _ := capnp.Unmarshal(d)
	o, _ := ReadRootCapnp2A(m)
	a.ID = o.Id()
	a.Tag = o.Tag()
	a.Timestamp = o.Timestamp()
	frames, _ := o.Frames()
	for i := 0; i < frames.Len(); i++ {
		a.Frames = append(a.Frames, frames.At(i))
	}
	a.GID = o.Gid()
	a.TxID = o.Txid()
	return nil
}

func (x *CapNProto2Serializer) String() string {
	return "CapNProto2"
}

func BenchmarkCapNProto2Marshal(b *testing.B) {
	benchMarshal(b, &CapNProto2Serializer{capnp.SingleSegment(nil)})
}

func BenchmarkCapNProto2Unmarshal(b *testing.B) {
	benchUnmarshal(b, &CapNProto2Serializer{capnp.SingleSegment(nil)})
}

// github.com/hprose/hprose-go/io

type HproseSerializer struct {
	writer *hprose.Writer
	reader *hprose.Reader
}

func (s *HproseSerializer) Marshal(o interface{}) []byte {
	a := o.(*A)
	writer := s.writer
	buf := writer.Stream.(*bytes.Buffer)
	l := buf.Len()
	writer.WriteInt64(a.ID)
	writer.WriteUint64(uint64(a.Tag))
	writer.WriteInt64(a.Timestamp)
	// TODO: frames
	writer.WriteInt64(a.GID)
	writer.WriteUint64(a.TxID)
	return buf.Bytes()[l:]
}

func (s *HproseSerializer) Unmarshal(d []byte, i interface{}) error {
	o := i.(*A)
	reader := s.reader
	reader.Stream = &hprose.BytesReader{d, 0}
	o.ID, _ = reader.ReadInt64()
	tag, _ := reader.ReadUint64()
	o.Tag = uint8(tag)
	o.Timestamp, _ = reader.ReadInt64()
	// TODO: frames
	o.GID, _ = reader.ReadInt64()
	o.TxID, _ = reader.ReadUint64()
	return nil
}

func (s *HproseSerializer) String() string {
	return "Hprose"
}

func BenchmarkHproseMarshal(b *testing.B) {
	buf := new(bytes.Buffer)
	writer := hprose.NewWriter(buf, true)
	benchMarshal(b, &HproseSerializer{writer: writer})
}

func BenchmarkHproseUnmarshal(b *testing.B) {
	buf := new(bytes.Buffer)
	reader := hprose.NewReader(buf, true)
	bufw := new(bytes.Buffer)
	writer := hprose.NewWriter(bufw, true)
	benchUnmarshal(b, &HproseSerializer{writer: writer, reader: reader})
}

// github.com/DeDiS/protobuf

type ProtobufSerializer struct{}

func (m ProtobufSerializer) Marshal(o interface{}) []byte {
	d, _ := protobuf.Encode(o)
	return d
}

func (m ProtobufSerializer) Unmarshal(d []byte, o interface{}) error {
	return protobuf.Decode(d, o)
}

func (m ProtobufSerializer) String() string {
	return "protobuf"
}

func BenchmarkProtobufMarshal(b *testing.B) {
	benchMarshal(b, ProtobufSerializer{})
}

func BenchmarkProtobufUnmarshal(b *testing.B) {
	benchUnmarshal(b, ProtobufSerializer{})
}

// github.com/golang/protobuf

func generateProto() []*ProtoBufA {
	a := make([]*ProtoBufA, 0, 1000)
	for i := 0; i < 1000; i++ {
		a = append(a, &ProtoBufA{
			ID:        proto.Int64(int64(rand.Intn(1000000))),
			Tag:       proto.Uint64(uint64(rand.Intn(256))),
			Timestamp: proto.Int64(int64(rand.Intn(1000000))),
			Frames: []uint64{
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
			},
			GID:  proto.Int64(int64(rand.Intn(1000000))),
			TxID: proto.Uint64(uint64(rand.Intn(1000000))),
		})
	}
	return a
}

func BenchmarkGoprotobufMarshal(b *testing.B) {
	b.StopTimer()
	data := generateProto()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(data[rand.Intn(len(data))])
	}
}

func BenchmarkGoprotobufUnmarshal(b *testing.B) {
	b.StopTimer()
	data := generateProto()
	ser := make([][]byte, len(data))
	for i, d := range data {
		ser[i], _ = proto.Marshal(d)
	}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n := rand.Intn(len(ser))
		o := &ProtoBufA{}
		err := proto.Unmarshal(ser[n], o)
		if err != nil {
			b.Fatalf("goprotobuf failed to unmarshal: %s (%s)", err, ser[n])
		}
		// Validate unmarshalled data.
		if validate != "" {
			i := data[n]
			correct := *i.ID == *o.ID &&
				*i.Tag == *o.Tag &&
				*i.Timestamp == *o.Timestamp &&
				compareUint64(i.Frames, o.Frames) &&
				*i.GID == *o.GID &&
				*i.TxID == *o.TxID
			if !correct {
				b.Fatalf("unmarshaled object differed:\n%v\n%v", i, o)
			}
		}
	}
}

// github.com/gogo/protobuf/proto

func generateGogoProto() []*GogoProtoBufA {
	a := make([]*GogoProtoBufA, 0, 1000)
	for i := 0; i < 1000; i++ {
		a = append(a, &GogoProtoBufA{
			ID:        int64(rand.Intn(1000000)),
			Tag:       uint64(rand.Intn(256)),
			Timestamp: int64(rand.Intn(1000000)),
			Frames: []uint64{
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
			},
			GID:  int64(rand.Intn(1000000)),
			TxID: uint64(rand.Intn(1000000)),
		})
	}
	return a
}

func BenchmarkGogoprotobufMarshal(b *testing.B) {
	b.StopTimer()
	data := generateGogoProto()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(data[rand.Intn(len(data))])
	}
}

func BenchmarkGogoprotobufUnmarshal(b *testing.B) {
	b.StopTimer()
	data := generateGogoProto()
	ser := make([][]byte, len(data))
	for i, d := range data {
		ser[i], _ = proto.Marshal(d)
	}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n := rand.Intn(len(ser))
		o := &GogoProtoBufA{}
		err := proto.Unmarshal(ser[n], o)
		if err != nil {
			b.Fatalf("goprotobuf failed to unmarshal: %s (%s)", err, ser[n])
		}
		// Validate unmarshalled data.
		if validate != "" {
			i := data[n]
			correct := i == o
			if !correct {
				b.Fatalf("unmarshaled object differed:\n%v\n%v", i, o)
			}
		}
	}
}

// github.com/andyleap/gencode

func generateGencode() []*GencodeA {
	a := make([]*GencodeA, 0, 1000)
	for i := 0; i < 1000; i++ {
		a = append(a, &GencodeA{
			ID:        int64(rand.Intn(1000000)),
			Tag:       uint8(rand.Intn(256)),
			Timestamp: int64(rand.Intn(1000000)),
			Frames: []uint64{
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
			},
			GID:  int64(rand.Intn(1000000)),
			TxID: uint64(rand.Intn(1000000)),
		})
	}
	return a
}

func BenchmarkGencodeMarshal(b *testing.B) {
	b.StopTimer()
	data := generateGencode()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		data[rand.Intn(len(data))].Marshal(nil)
	}
}

func BenchmarkGencodeUnmarshal(b *testing.B) {
	b.StopTimer()
	data := generateGencode()
	ser := make([][]byte, len(data))
	for i, d := range data {
		ser[i], _ = d.Marshal(nil)
	}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n := rand.Intn(len(ser))
		o := &GencodeA{}
		_, err := o.Unmarshal(ser[n])
		if err != nil {
			b.Fatalf("gencode failed to unmarshal: %s (%s)", err, ser[n])
		}
		// Validate unmarshalled data.
		if validate != "" {
			i := data[n]
			correct := i == o
			if !correct {
				b.Fatalf("unmarshaled object differed:\n%v\n%v", i, o)
			}
		}
	}
}

func generateGencodeUnsafe() []*GencodeUnsafeA {
	a := make([]*GencodeUnsafeA, 0, 1000)
	for i := 0; i < 1000; i++ {
		a = append(a, &GencodeUnsafeA{
			ID:        int64(rand.Intn(1000000)),
			Tag:       uint8(rand.Intn(256)),
			Timestamp: int64(rand.Intn(1000000)),
			Frames: []uint64{
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
				uint64(rand.Intn(1000000)),
			},
			GID:  int64(rand.Intn(1000000)),
			TxID: uint64(rand.Intn(1000000)),
		})
	}
	return a
}

func BenchmarkGencodeUnsafeMarshal(b *testing.B) {
	b.StopTimer()
	data := generateGencodeUnsafe()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		data[rand.Intn(len(data))].Marshal(nil)
	}
}

func BenchmarkGencodeUnsafeUnmarshal(b *testing.B) {
	b.StopTimer()
	data := generateGencodeUnsafe()
	ser := make([][]byte, len(data))
	for i, d := range data {
		ser[i], _ = d.Marshal(nil)
	}
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n := rand.Intn(len(ser))
		o := &GencodeUnsafeA{}
		_, err := o.Unmarshal(ser[n])
		if err != nil {
			b.Fatalf("gencode failed to unmarshal: %s (%s)", err, ser[n])
		}
		// Validate unmarshalled data.
		if validate != "" {
			i := data[n]
			correct := i == o
			if !correct {
				b.Fatalf("unmarshaled object differed:\n%v\n%v", i, o)
			}
		}
	}
}
