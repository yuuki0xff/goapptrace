package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
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

// Create a temporary file, and returns File object.
// You should remove a temporary file after used.
func createTempFile() File {
	file, err := ioutil.TempFile("", ".goapptrace_storage")
	if err != nil {
		panic(err)
	}
	return File(file.Name())
}

func must(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatal(msg, err)
	}
}

func TestEncoderDecoder(t *testing.T) {
	var enc Encoder
	var dec Decoder

	tmpfile := createTempFile()
	defer func() {
		must(t, os.Remove(string(tmpfile)), "os.Remove:")
	}()

	// test Encoder's
	enc = Encoder{
		File: tmpfile,
	}
	must(t, enc.Open(), "enc.Open:")
	must(t, enc.Append(testData1), "enc.Append 1:")
	must(t, enc.Append(testData2), "enc.Append 2:")
	must(t, enc.Append(testData3), "enc.Append 3:")
	must(t, enc.Close(), "enc.Close():")

	// test Decoder.Open()/Close()/Read() methods
	var data1 testStruct1
	var data2 testStruct2
	var data3 testStruct3
	dec = Decoder{
		File: tmpfile,
	}
	must(t, dec.Open(), "dec.Open:")
	must(t, dec.Read(&data1), "dec.Read 1:")
	if data1 != testData1 {
		t.Fatalf("Miss match data: expect %+v, but got %+v", testData1, data1)
	}
	must(t, dec.Read(&data2), "dec.Read 2:")
	if data2 != testData2 {
		t.Fatalf("Miss match data: expect %+v, but got %+v", testData2, data2)
	}
	must(t, dec.Read(&data3), "dec.Read 3:")
	if data3 != testData3Result {
		t.Fatalf("Miss match data: expect %+v, but got %+v", testData3Result, data3)
	}
	must(t, dec.Close(), "dec.Close:")
}

func TestDecoder_Walk(t *testing.T) {
	var enc Encoder
	var dec Decoder

	tmpfile := createTempFile()
	defer func() {
		must(t, os.Remove(string(tmpfile)), "os.Remove:")
	}()

	// test Encoder's
	enc = Encoder{
		File: tmpfile,
	}
	must(t, enc.Open(), "enc.Open:")
	must(t, enc.Append(testStruct2{"data1"}), "enc.Append 1:")
	must(t, enc.Append(testStruct2{"data2"}), "enc.Append 2:")
	must(t, enc.Append(testStruct2{"data3"}), "enc.Append 3:")
	must(t, enc.Close(), "enc.Close():")

	dec = Decoder{
		File: tmpfile,
	}
	i := 1
	must(t, dec.Open(), "dec.Open:")
	must(t, dec.Walk(
		func() interface{} {
			return &testStruct2{}
		},
		func(data interface{}) error {
			t.Log("received", data)
			expectData := &testStruct2{fmt.Sprintf("data%d", i)}
			decodedData, ok := data.(*testStruct2)
			if !ok {
				t.Fatalf("Miss match data type: expect %+v, but got %+v", expectData, decodedData)
			}
			if *expectData != *decodedData {
				t.Fatalf("Miss match data: expect %+v, but got %+v", expectData, decodedData)
			}

			i++
			return nil
		},
	), "dec.Walk:")
	if 3 != i-1 {
		t.Fatalf("expect receive 3 data, but %d data", i-1)
	}
	must(t, dec.Close(), "dec.Close:")
}
