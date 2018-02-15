package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

var (
	uint64Value1 = uint64(0x123456789abcdef0)
	uint64Bytes1 = []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	uint64Value2 = uint64(0x1234000000000000)
	uint64Bytes2 = []byte{0x12, 0x34, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	uint64Value3 = uint64(0x00000000000000ff)
	uint64Bytes3 = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}

	funcSymbol = &logutil.FuncSymbol{
		ID:    0xa00000000000000a,
		Name:  "name",
		File:  "file path",
		Entry: 0xb00000000000000b,
	}
	funcSymbolBytes = []byte{
		// ID
		0xa0, 0, 0, 0, 0, 0, 0, 0x0a,
		// Name: string len
		0, 0, 0, 0, 0, 0, 0, 4,
		// Name: string body
		0x6e, 0x61, 0x6d, 0x65,
		// File: string len
		0, 0, 0, 0, 0, 0, 0, 9,
		// File: string body
		0x66, 0x69, 0x6c, 0x65, 0x20, 0x70, 0x61, 0x74, 0x68,
		// Entry
		0xb0, 0, 0, 0, 0, 0, 0, 0x0b,
	}

	funcStatus = &logutil.FuncStatus{
		ID:   0xa00000000000000a,
		Func: 0xb00000000000000b,
		Line: 0xc00000000000000c,
		PC:   0xd00000000000000d,
	}
	funcStatusBytes = []byte{
		// ID
		0xa0, 0, 0, 0, 0, 0, 0, 0x0a,
		// Func ID
		0xb0, 0, 0, 0, 0, 0, 0, 0x0b,
		// Line
		0xc0, 0, 0, 0, 0, 0, 0, 0x0c,
		// PC
		0xd0, 0, 0, 0, 0, 0, 0, 0x0d,
	}

	rawFuncLog = &logutil.RawFuncLog{
		ID:        logutil.RawFuncLogID(0x0a0000000000000a),
		Tag:       "tag name",
		Timestamp: logutil.Time(0x0b0000000000000b),
		Frames: []logutil.FuncStatusID{
			1, 2, 3,
		},
		GID:  logutil.GID(0x0c0000000000000c),
		TxID: logutil.TxID(0x0d0000000000000d),
	}
	rawFuncLogBytes = []byte{
		// ID
		0x0a, 0, 0, 0, 0, 0, 0, 0x0a,
		// Tag: string len
		0, 0, 0, 0, 0, 0, 0, 8,
		// Tag: string body
		0x74, 0x61, 0x67, 0x20, 0x6e, 0x61, 0x6d, 0x65,
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
	w := &fakeWriter{}
	val := true
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalBool(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalBool(b *testing.B) {
	r := &fakeReader{
		B: []byte{1},
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalBool(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalUint64(b *testing.B) {
	w := &fakeWriter{}
	val := uint64(10)
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalUint64(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalUint64(b *testing.B) {
	r := &fakeReader{
		B: []byte{0, 0, 0, 0, 0, 0, 0, 0xa},
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalUint64(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalString(b *testing.B) {
	w := &fakeWriter{}
	val := "test string"
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalString(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalString(b *testing.B) {
	r := &fakeReader{
		B: []byte{
			// length
			0, 0, 0, 0, 0, 0, 0, 0xb,
			// string data
			0x66, 0x61, 0x6b, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
		},
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalString(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalFuncSymbol(b *testing.B) {
	w := &fakeWriter{}
	val := funcSymbol
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalFuncSymbol(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalFuncSymbol(b *testing.B) {
	r := &fakeReader{
		B: funcSymbolBytes,
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalFuncSymbol(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalFuncStatus(b *testing.B) {
	w := &fakeWriter{}
	val := funcStatus
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalFuncStatus(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalFuncStatus(b *testing.B) {
	r := &fakeReader{
		B: funcStatusBytes,
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalFuncStatus(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalFuncStatusIDSlice(b *testing.B) {
	w := &fakeWriter{}
	val := []logutil.FuncStatusID{1, 2, 3, 4, 5, 6, 8, 9, 10}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalFuncStatusIDSlice(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalFuncStatusIDSlice(b *testing.B) {
	var w bytes.Buffer
	marshalFuncStatusIDSlice(&w, []logutil.FuncStatusID{1, 2, 3, 4, 5, 6, 8, 9, 10})
	r := &fakeReader{
		B: w.Bytes(),
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalFuncStatus(r)
		r.N = 0
	}
	b.StopTimer()
}
func BenchmarkMarshalRawFuncLog(b *testing.B) {
	w := &fakeWriter{}
	val := rawFuncLog
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		marshalRawFuncLog(w, val)
	}
	b.StopTimer()
}
func BenchmarkUnmarshalRawFuncLog(b *testing.B) {
	r := &fakeReader{
		B: rawFuncLogBytes,
	}
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		unmarshalRawFuncLog(r)
		r.N = 0
	}
	b.StopTimer()
}

func TestMarshalBool(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	marshalBool(&buf, false)
	marshalBool(&buf, true)
	a.Equal([]byte{0, 1}, buf.Bytes())
}
func TestUnmarshalBool(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	buf.WriteByte(0)
	buf.WriteByte(1)

	a.False(unmarshalBool(&buf))
	a.True(unmarshalBool(&buf))
}
func TestMarshalUint64(t *testing.T) {
	a := assert.New(t)
	test := func(val uint64, b []byte) {
		var buf bytes.Buffer
		marshalUint64(&buf, val)
		a.Len(buf.Bytes(), 8)
		a.Equal(b, buf.Bytes())
	}

	test(uint64Value1, uint64Bytes1)
	test(uint64Value2, uint64Bytes2)
	test(uint64Value3, uint64Bytes3)
}
func TestUnmarshalUint64(t *testing.T) {
	a := assert.New(t)
	test := func(val uint64, b []byte) {
		var buf bytes.Buffer
		buf.Write(b)
		a.Equal(val, unmarshalUint64(&buf))
	}

	test(uint64Value1, uint64Bytes1)
	test(uint64Value2, uint64Bytes2)
	test(uint64Value3, uint64Bytes3)
}
func TestMarshalString(t *testing.T) {
	a := assert.New(t)
	test := func(msg, s string, blen, bstr []byte) {
		var buf bytes.Buffer
		marshalString(&buf, s)
		a.Equal(blen, buf.Bytes()[:8], msg+": length field")
		a.Equal(bstr, buf.Bytes()[8:], msg+": string field")
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
	test := func(msg, s string, blen, bstr []byte) {
		var buf bytes.Buffer
		buf.Write(blen)
		buf.Write(bstr)
		a.Equal(s, unmarshalString(&buf), msg)
	}

	test("empty string", "",
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{})
	test("non-empty string", "foo bar",
		[]byte{0, 0, 0, 0, 0, 0, 0, 7},
		[]byte{0x66, 0x6f, 0x6f, 0x20, 0x62, 0x61, 0x72})
}
func TestMarshalFuncSymbolSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []*logutil.FuncSymbol, sliceLen, nonNilFlag, fsBytes []byte) {
		var buf bytes.Buffer
		marshalFuncSymbolSlice(&buf, slice)
		a.Equal(sliceLen, buf.Bytes()[:8], msg+": length field")
		if nonNilFlag == nil {
			return
		}
		a.Equal(nonNilFlag, buf.Bytes()[8:9], msg+": non-nil flag field")
		if fsBytes == nil {
			return
		}
		a.Equal(fsBytes, buf.Bytes()[9:], msg+": FuncSymbol field")
	}

	test("empty slice",
		[]*logutil.FuncSymbol{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil, nil)
	test("contains a nil item",
		[]*logutil.FuncSymbol{nil},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0}, nil)
	test("contains a non-nil item",
		[]*logutil.FuncSymbol{funcSymbol},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{1},
		funcSymbolBytes)
}
func TestUnmarshalFuncSymbolSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []*logutil.FuncSymbol, sliceLen, nonNilFlag, fsBytes []byte) {
		var buf bytes.Buffer
		buf.Write(sliceLen)
		buf.Write(nonNilFlag)
		buf.Write(fsBytes)
		a.Equal(slice, unmarshalFuncSymbolSlice(&buf), msg)
	}

	test("empty",
		[]*logutil.FuncSymbol{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil, nil)
	test("contains a nil item",
		[]*logutil.FuncSymbol{nil},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0}, nil)
	test("contains a non-nil item",
		[]*logutil.FuncSymbol{funcSymbol},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{1},
		funcSymbolBytes)
}
func TestMarshalFuncSymbol(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	marshalFuncSymbol(&buf, funcSymbol)
	a.Equal(funcSymbolBytes, buf.Bytes())
}
func TestUnmarshalFuncSymbol(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	buf.Write(funcSymbolBytes)
	a.Equal(funcSymbol, unmarshalFuncSymbol(&buf))
}
func TestMarshalFuncStatusSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []*logutil.FuncStatus, sliceLen, nonNilFlag, fsBytes []byte) {
		var buf bytes.Buffer
		marshalFuncStatusSlice(&buf, slice)
		a.Equal(sliceLen, buf.Bytes()[:8], msg+": length field")
		if nonNilFlag == nil {
			return
		}
		a.Equal(nonNilFlag, buf.Bytes()[8:9], msg+": non-nil flag field")
		if fsBytes == nil {
			return
		}
		a.Equal(fsBytes, buf.Bytes()[9:], msg+": FuncStatus field")
	}

	test("empty slice",
		[]*logutil.FuncStatus{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil, nil)
	test("contains a nil item",
		[]*logutil.FuncStatus{nil},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0}, nil)
	test("contains a non-nil item",
		[]*logutil.FuncStatus{funcStatus},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{1},
		funcStatusBytes)
}
func TestUnmarshalFuncStatusSlice(t *testing.T) {
	a := assert.New(t)
	test := func(msg string, slice []*logutil.FuncStatus, sliceLen, nonNilFlag, fsBytes []byte) {
		var buf bytes.Buffer
		buf.Write(sliceLen)
		buf.Write(nonNilFlag)
		buf.Write(fsBytes)
		a.Equal(slice, unmarshalFuncStatusSlice(&buf), msg)
	}

	test("empty",
		[]*logutil.FuncStatus{},
		[]byte{0, 0, 0, 0, 0, 0, 0, 0},
		nil, nil)
	test("contains a nil item",
		[]*logutil.FuncStatus{nil},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0}, nil)
	test("contains a non-nil item",
		[]*logutil.FuncStatus{funcStatus},
		[]byte{0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{1},
		funcStatusBytes)
}
func TestMarshalFuncStatus(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	marshalFuncStatus(&buf, funcStatus)
	a.Equal(funcStatusBytes, buf.Bytes())
}
func TestUnmarshalFuncStatus(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	buf.Write(funcStatusBytes)
	a.Equal(funcStatus, unmarshalFuncStatus(&buf))
}
func TestMarshalRawFuncLog(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	marshalRawFuncLog(&buf, rawFuncLog)
	a.Equal(rawFuncLogBytes, buf.Bytes())
}
func TestUnmarshalRawFuncLog(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	buf.Write(rawFuncLogBytes)
	a.Equal(rawFuncLog, unmarshalRawFuncLog(&buf))
}
