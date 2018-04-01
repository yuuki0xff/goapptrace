package storage

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/util"
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

func TestStore_WriteNolock(t *testing.T) {
	t.Run("sequential write", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			var data [65536]uint64
			for i := range data {
				data[i] = uint64(i)
			}

			s := Store{
				File:       File(tmpfile),
				RecordSize: 8,
			}
			a.NoError(s.Open())
			s.Lock()
			for i := range data {
				a.NoError(s.WriteNolock(int64(i), func(buf []byte) int64 {
					binary.LittleEndian.PutUint64(buf, data[i])
					return 8
				}))
			}

			for i := range data {
				a.NoError(s.ReadNolock(int64(i), func(buf []byte) {
					actual := binary.LittleEndian.Uint64(buf)
					if data[i] != actual {
						a.Equal(data[i], actual)
					}
				}))
			}
			s.Unlock()
			a.NoError(s.Close())
		})
	})

	t.Run("random write", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			var data [65536]uint64
			for i := range data {
				data[i] = uint64(i)
			}

			var randIdx [65536]int
			for i := range randIdx {
				randIdx[i] = rand.Intn(len(data))
			}

			s := Store{
				File:       File(tmpfile),
				RecordSize: 8,
			}
			a.NoError(s.Open())
			s.Lock()
			for i := range randIdx {
				idx := randIdx[i]
				a.NoError(s.WriteNolock(int64(idx), func(buf []byte) int64 {
					binary.LittleEndian.PutUint64(buf, data[idx])
					return 8
				}))
			}

			for i := range randIdx {
				idx := randIdx[i]
				a.NoError(s.ReadNolock(int64(idx), func(buf []byte) {
					actual := binary.LittleEndian.Uint64(buf)
					if data[idx] != actual {
						a.Equal(data[idx], actual)
					}
				}))
			}
			s.Unlock()
			a.NoError(s.Close())
		})
	})
}
