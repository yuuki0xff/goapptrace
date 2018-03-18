package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func ExampleStore() {
	f, _ := ioutil.TempFile("", "")
	f.Close()
	defer os.Remove(f.Name())

	s := Store{
		File:       File(f.Name()),
		RecordSize: 20,
		ReadOnly:   false,
	}
	s.Open()
	defer s.Close()
	s.Append(func(buf []byte) int64 {
		return int64(copy(buf, []byte("hello world")))
	})

	s.Read(0, func(buf []byte) {
		fmt.Print(string(buf[:11]))
	})
	// Output: hello world
}
func ExampleStore_Lock() {
	f, _ := ioutil.TempFile("", "")
	f.Close()
	defer os.Remove(f.Name())

	s := Store{
		File:       File(f.Name()),
		RecordSize: 10,
		ReadOnly:   false,
	}
	s.Open()
	defer s.Close()

	s.Lock()
	s.AppendNolock(func(buf []byte) int64 {
		return int64(copy(buf, []byte("world")))
	})
	s.AppendNolock(func(buf []byte) int64 {
		return int64(copy(buf, []byte("hello")))
	})
	s.ReadNolock(1, func(buf []byte) {
		fmt.Print(string(buf[:5]))
	})
	s.ReadNolock(0, func(buf []byte) {
		fmt.Print(string(buf[:5]))
	})
	s.Unlock()
	// Output: helloworld
}

func BenchmarkStore_Append(b *testing.B) {
	s := Store{
		File:       File(os.DevNull),
		RecordSize: 10,
		ReadOnly:   false,
	}
	s.Open()
	defer s.Close()
	data := []byte("hello")
	writer := func(buf []byte) int64 {
		copy(buf, data)
		return 10
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Append(writer)
	}
	b.StopTimer()
}
func BenchmarkStore_AppendNolock(b *testing.B) {
	s := Store{
		File:       File(os.DevNull),
		RecordSize: 10,
		ReadOnly:   false,
	}
	s.Open()
	defer s.Close()
	data := []byte("hello")
	writer := func(buf []byte) int64 {
		copy(buf, data)
		return 10
	}

	b.ResetTimer()
	s.Lock()
	for i := 0; i < b.N; i++ {
		s.AppendNolock(writer)
	}
	s.Unlock()
	b.StopTimer()
}
