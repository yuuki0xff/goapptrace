package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

var (
	uint64Value1 = uint64(0x123456789abcdef0)
	uint64Bytes1 = []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	uint64Value2 = uint64(0x1234000000000000)
	uint64Bytes2 = []byte{0x12, 0x34, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	uint64Value3 = uint64(0x00000000000000ff)
	uint64Bytes3 = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}

	goFunc = &types.GoFunc{
		Name:  "name",
		Entry: 0xb00000000000000b,
	}
	goFuncBytes = []byte{
		// Name: string len
		0, 0, 0, 0, 0, 0, 0, 4,
		// Name: string body
		0x6e, 0x61, 0x6d, 0x65,
		// Entry
		0xb0, 0, 0, 0, 0, 0, 0, 0x0b,
	}

	goLine = &types.GoLine{
		PC:     0xd00000000000000d,
		FileID: 0xb000000c,
		Line:   0xa000000c,
	}
	goLineBytes = []byte{
		// PC
		0xd0, 0, 0, 0, 0, 0, 0, 0x0d,
		// FileID
		0xb0, 0x00, 0x00, 0x0c,
		// Line
		0xa0, 0, 0, 0x0c,
	}

	rawFuncLog = &types.RawFuncLog{
		ID:        types.RawFuncLogID(0x0a0000000000000a),
		Tag:       types.FuncEnd,
		Timestamp: types.Time(0x0b0000000000000b),
		Frames: []uintptr{
			1, 2, 3,
		},
		GID:  types.GID(0x0c0000000000000c),
		TxID: types.TxID(0x0d0000000000000d),
	}
	rawFuncLogBytes = []byte{
		// ID
		0x0a, 0, 0, 0, 0, 0, 0, 0x0a,
		// Tag: FuncEnd
		1,
		// Timestamp
		0x0b, 0, 0, 0, 0, 0, 0, 0x0b,
		// Frames: slice len
		0, 0, 0, 0, 0, 0, 0, 3,
		// Frames: slice body
		0, 0, 0, 0, 0, 0, 0, 1,
		0, 0, 0, 0, 0, 0, 0, 2,
		0, 0, 0, 0, 0, 0, 0, 3,
		// GID
		0x0c, 0, 0, 0, 0, 0, 0, 0x0c,
		// TxID
		0x0d, 0, 0, 0, 0, 0, 0, 0x0d,
	}
)

func BenchmarkMarshalBool(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := true
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalBool(buf, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalBool(b *testing.B) {
	buf := []byte{1}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalBool(buf)
	}
	b.StopTimer()
}
func BenchmarkMarshalUint64(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := uint64(10)
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalUint64(buf, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalUint64(b *testing.B) {
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0xa}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalUint64(buf)
	}
	b.StopTimer()
}
func BenchmarkMarshalString(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := "test string"
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalString(buf, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalString(b *testing.B) {
	buf := []byte{
		// length
		0, 0, 0, 0, 0, 0, 0, 0xb,
		// string data
		0x66, 0x61, 0x6b, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalString(buf)
	}
	b.StopTimer()
}
func BenchmarkMarshalGoFunc(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := goFunc
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalGoFunc(buf, *val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalGoFunc(b *testing.B) {
	buf := goFuncBytes
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalGoFunc(buf)
	}
	b.StopTimer()
}
func BenchmarkMarshalGoLine(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := goLine
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalGoLine(buf, *val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalGoLine(b *testing.B) {
	buf := goLineBytes
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalGoLine(buf)
	}
	b.StopTimer()
}
func BenchmarkMarshalUintptrSlice(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := []uintptr{1, 2, 3, 4, 5, 6, 8, 9, 10}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalUintptrSlice(buf, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalUintptrSlice(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	n := marshalUintptrSlice(buf, []uintptr{1, 2, 3, 4, 5, 6, 8, 9, 10})
	buf = buf[:n]

	slice := make([]uintptr, 100)
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		s := slice
		unmarshalUintptrSlice(buf, &s)
	}
	b.StopTimer()
}
func BenchmarkMarshalRawFuncLog(b *testing.B) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	val := rawFuncLog
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalRawFuncLog(buf, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalRawFuncLog(b *testing.B) {
	buf := rawFuncLogBytes
	var fl types.RawFuncLog
	fl.Frames = make([]uintptr, 100)
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalRawFuncLog(buf, &fl)
	}
	b.StopTimer()
}

func TestMarshalBool(t *testing.T) {
	buf := make([]byte, 2)
	a := assert.New(t)

	total := marshalBool(buf, false)
	total += marshalBool(buf[total:], true)
	a.Equal([]byte{0, 1}, buf[:total])
}
func TestUnmarshalBool(t *testing.T) {
	buf := []byte{0, 1}
	a := assert.New(t)

	val1, _ := unmarshalBool(buf[:1])
	val2, _ := unmarshalBool(buf[1:])
	a.False(val1)
	a.True(val2)
}
func TestMarshalUint64(t *testing.T) {
	a := assert.New(t)
	test := func(val uint64, b []byte) {
		buf := make([]byte, DefaultMaxSmallPacketSize)
		n := marshalUint64(buf, val)
		buf = buf[:n]

		a.Len(buf, 8)
		a.Equal(b, buf)
	}

	test(uint64Value1, uint64Bytes1)
	test(uint64Value2, uint64Bytes2)
	test(uint64Value3, uint64Bytes3)
}
func TestUnmarshalUint64(t *testing.T) {
	a := assert.New(t)
	test := func(expected uint64, b []byte) {
		actual, _ := unmarshalUint64(b)
		a.Equal(expected, actual)
	}

	test(uint64Value1, uint64Bytes1)
	test(uint64Value2, uint64Bytes2)
	test(uint64Value3, uint64Bytes3)
}
func TestMarshalString(t *testing.T) {
	a := assert.New(t)
	test := func(msg, s string, blen, bstr []byte) {
		buf := make([]byte, DefaultMaxSmallPacketSize)
		n := marshalString(buf, s)
		buf = buf[:n]
		a.Equal(blen, buf[:8], msg+": length field")
		a.Equal(bstr, buf[8:], msg+": string field")
	}

	test("empty string", "",
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{})
	test("non-empty string", "test",
		[]byte{0, 0, 0, 0, 0, 0, 0, 4},
		[]byte{0x74, 0x65, 0x73, 0x74})
}
func TestUnmarshalString(t *testing.T) {
	a := assert.New(t)
	test := func(msg, expected string, blen, bstr []byte) {
		var buf bytes.Buffer
		buf.Write(blen)
		buf.Write(bstr)
		actual, _ := unmarshalString(buf.Bytes())
		a.Equal(expected, actual, msg)
	}

	test("empty string", "",
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{})
	test("non-empty string", "foo bar",
		[]byte{0, 0, 0, 0, 0, 0, 0, 7},
		[]byte{0x66, 0x6f, 0x6f, 0x20, 0x62, 0x61, 0x72})
}
func TestMarshalGoFuncSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []types.GoFunc, sliceLen, golineBytes []byte) {
		buf := make([]byte, DefaultMaxSmallPacketSize)
		n := marshalGoFuncSlice(buf, slice)
		buf = buf[:n]

		a.Equal(sliceLen, buf[:8], msg+": length field")
		if golineBytes == nil {
			return
		}
		a.Equal(golineBytes, buf[8:], msg+": GoFunc field")
	}

	test("empty slice",
		[]types.GoFunc{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil)
	test("contains a item",
		[]types.GoFunc{*goFunc},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		goFuncBytes)
}
func TestUnmarshalGoFuncSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, expected []types.GoFunc, sliceLen, gofuncBytes []byte) {
		var buf bytes.Buffer
		buf.Write(sliceLen)
		buf.Write(gofuncBytes)
		actual, _ := unmarshalGoFuncSlice(buf.Bytes())
		a.Equal(expected, actual, msg)
	}

	test("empty",
		[]types.GoFunc{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil)
	test("contains a item",
		[]types.GoFunc{*goFunc},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		goFuncBytes)
}
func TestMarshalGoFunc(t *testing.T) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	a := assert.New(t)

	n := marshalGoFunc(buf, *goFunc)
	buf = buf[:n]
	a.Equal(goFuncBytes, buf)
}
func TestUnmarshalGoFunc(t *testing.T) {
	a := assert.New(t)

	s, _ := unmarshalGoFunc(goFuncBytes)
	a.Equal(*goFunc, s)
}
func TestMarshalGoLineSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []types.GoLine, sliceLen, golineBytes []byte) {
		buf := make([]byte, DefaultMaxSmallPacketSize)
		n := marshalGoLineSlice(buf, slice)
		buf = buf[:n]

		a.Equal(sliceLen, buf[:8], msg+": length field")
		if golineBytes == nil {
			return
		}
		a.Equal(golineBytes, buf[8:], msg+": GoLine field")
	}

	test("empty slice",
		[]types.GoLine{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil)
	test("contains a item",
		[]types.GoLine{*goLine},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		goLineBytes)
}
func TestUnmarshalGoLineSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, expected []types.GoLine, sliceLen, golineBytes []byte) {
		var buf bytes.Buffer
		buf.Write(sliceLen)
		buf.Write(golineBytes)

		actual, _ := unmarshalGoLineSlice(buf.Bytes())
		a.Equal(expected, actual, msg)
	}

	test("empty",
		[]types.GoLine{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil)
	test("contains item",
		[]types.GoLine{*goLine},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		goLineBytes)
}
func TestMarshalGoLine(t *testing.T) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	a := assert.New(t)

	n := marshalGoLine(buf, *goLine)
	buf = buf[:n]
	a.Equal(goLineBytes, buf)
}
func TestUnmarshalGoLine(t *testing.T) {
	a := assert.New(t)

	fs, _ := unmarshalGoLine(goLineBytes)
	a.Equal(*goLine, fs)
}
func TestMarshalRawFuncLog(t *testing.T) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	a := assert.New(t)

	n := marshalRawFuncLog(buf, rawFuncLog)
	buf = buf[:n]
	a.Equal(rawFuncLogBytes, buf)
}
func TestUnmarshalRawFuncLog(t *testing.T) {
	a := assert.New(t)

	var fl types.RawFuncLog
	fl.Frames = make([]uintptr, 100)
	unmarshalRawFuncLog(rawFuncLogBytes, &fl)
	a.Equal(rawFuncLog, &fl)
}
