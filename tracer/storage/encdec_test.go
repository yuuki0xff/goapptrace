package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

var (
	testData1       = testStruct1{}
	testData2       = testStruct2{"2"}
	testData3       = testStruct3{3, 3}
	testData3Result = testStruct3{3, 0} // private fields can not saved.
)

type testStruct1 struct{}
type testStruct2 struct {
	PubField string
}
type testStruct3 struct {
	PrvField int
	prvField int
}

func TestEncoderDecoder(t *testing.T) {
	a := assert.New(t)

	util.WithTempFile(func(tmpfile string) {
		// test Encoder's
		enc := Encoder{
			File: File(tmpfile),
		}
		a.NoError(enc.Open())
		a.NoError(enc.Append(testData1))
		a.NoError(enc.Append(testData2))
		a.NoError(enc.Append(testData3))
		a.NoError(enc.Close())

		// test Decoder.Open()/Close()/Read() methods
		var data1 testStruct1
		var data2 testStruct2
		var data3 testStruct3
		dec := Decoder{
			File: File(tmpfile),
		}
		a.NoError(dec.Open())
		a.NoError(dec.Read(&data1))
		a.Equal(testData1, data1)

		a.NoError(dec.Read(&data2))
		a.Equal(testData2, data2)

		a.NoError(dec.Read(&data3))
		a.Equal(testData3Result, data3)
		a.NoError(dec.Close())
	})
}

func TestDecoder_Walk(t *testing.T) {
	a := assert.New(t)

	util.WithTempFile(func(tmpfile string) {
		// test Encoder's
		enc := Encoder{
			File: File(tmpfile),
		}
		a.NoError(enc.Open())
		a.NoError(enc.Append(testStruct2{"data1"}))
		a.NoError(enc.Append(testStruct2{"data2"}))
		a.NoError(enc.Append(testStruct2{"data3"}))
		a.NoError(enc.Close())

		dec := Decoder{
			File: File(tmpfile),
		}
		i := 1
		a.NoError(dec.Open())
		a.NoError(dec.Walk(
			func() interface{} {
				return &testStruct2{}
			},
			func(data interface{}) error {
				t.Log("received", data)
				expectData := &testStruct2{fmt.Sprintf("data%d", i)}
				a.Equal(expectData, data)
				i++
				return nil
			},
		))
		a.Equal(3, i-1)
		a.NoError(dec.Close())
	})
}
