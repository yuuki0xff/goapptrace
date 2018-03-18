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
		RecordSize: 10,
		ReadOnly:   false,
	}
	s.Open()
	defer s.Close()
	s.Append(func(buf []byte) int {
		return copy(buf, []byte("world"))
	})
	s.Append(func(buf []byte) int {
		return copy(buf, []byte("hello"))
	})

	s.Read(1, func(buf []byte) {
		fmt.Print(string(buf[:5]))
	})
	s.Read(0, func(buf []byte) {
		fmt.Print(string(buf[:5]))
	})
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
	writer := func(buf []byte) int {
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
	writer := func(buf []byte) int {
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
