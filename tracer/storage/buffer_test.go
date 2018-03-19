package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

func ExampleWriteBuffer() {
	f, _ := ioutil.TempFile("", "")
	defer os.Remove(f.Name())
	defer f.Close()
	wb := WriteBuffer{
		W:            f,
		MaxWriteSize: 10,
		BufferSize:   50,
	}

	wb.Seek(5)
	n := copy(wb.WriteBuffer(), []byte("World"))
	wb.Write(n)

	wb.Seek(0)
	n = copy(wb.WriteBuffer(), []byte("Hello"))
	wb.Write(n)

	wb.Flush()

	data, _ := ioutil.ReadFile(f.Name())
	fmt.Println(string(data))
	// Output: HelloWorld
}

func BenchmarkWriteBuffer_Write(b *testing.B) {
	w, err := File(os.DevNull).OpenWriteOnly()
	if err != nil {
		panic(err)
	}
	defer w.Close()
	wb := WriteBuffer{
		W:            w,
		MaxWriteSize: 10,
		BufferSize:   1 << 12,
	}
	defer wb.Flush()
	data := []byte("helloworld")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.Write(copy(wb.WriteBuffer(), data))
	}
	b.StopTimer()
}
func BenchmarkWriteBuffer_Seek(b *testing.B) {
	w, err := File(os.DevNull).OpenWriteOnly()
	if err != nil {
		panic(err)
	}
	defer w.Close()
	wb := WriteBuffer{
		W:            w,
		MaxWriteSize: 10,
		BufferSize:   1 << 12,
	}
	defer wb.Flush()

	b.Run("seek-only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wb.Seek(int64(15 * i))
		}
		b.StopTimer()
	})

	b.Run("write-seek", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wb.WriteBuffer()
			wb.Write(10)
			wb.Seek(int64(15 * i))
		}
		b.StopTimer()
	})
}
func BenchmarkNoBuffer_Write(b *testing.B) {
	w, err := File(os.DevNull).OpenWriteOnly()
	if err != nil {
		panic(err)
	}
	defer w.Close()
	data := make([]byte, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write(data)
	}
	b.StopTimer()
}

type wbTestHelper struct {
	t    *testing.T
	name string
	fn   func(helper *wbTestHelperArgs)
}
type wbTestHelperArgs struct {
	t       *testing.T
	a       *assert.Assertions
	tmpfile string
	reader  FileReader
	writer  FileWriter
	readAll func() []byte
}

func (h wbTestHelper) Run() {
	util.WithTempFile(func(tmpfile string) {
		r, err := File(tmpfile).OpenReadOnly()
		if err != nil {
			panic(err)
		}
		defer r.Close()

		w, err := File(tmpfile).OpenWriteOnly()
		if err != nil {
			panic(err)
		}
		defer w.Close()

		h.t.Run(h.name, func(t *testing.T) {
			newh := wbTestHelperArgs{
				t:       t,
				a:       assert.New(t),
				tmpfile: tmpfile,
				reader:  r,
				writer:  w,
				readAll: func() []byte {
					data, err := ioutil.ReadFile(tmpfile)
					if err != nil {
						panic(err)
					}
					return data
				},
			}
			h.fn(&newh)
		})
	})
}

func TestWriteBuffer_WriteBuffer(t *testing.T) {
	wbTestHelper{
		t:    t,
		name: "validation",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 10,
				BufferSize:   0,
			}
			a.Panics(func() {
				wb.WriteBuffer()
			})
		},
	}.Run()

	wbTestHelper{
		t:    t,
		name: "ok",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 10,
				BufferSize:   50,
			}
			buf := wb.WriteBuffer()
			a.NotNil(buf)
			a.Len(buf, 10)
			a.Equal(50, cap(buf))
		},
	}.Run()
}
func TestWriteBuffer_Write(t *testing.T) {
	wbTestHelper{
		t:    t,
		name: "validation",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 6,
				BufferSize:   20,
			}
			a.Panics(func() {
				wb.Write(100)
			})
		},
	}.Run()

	wbTestHelper{
		t:    t,
		name: "normal",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 6,
				BufferSize:   20,
			}
			a.NoError(wb.Write(copy(wb.WriteBuffer(), []byte{1})))
			a.NoError(wb.Write(copy(wb.WriteBuffer(), []byte{2, 3})))
			a.NoError(wb.Write(copy(wb.WriteBuffer(), []byte{4, 5, 6})))
			a.NoError(wb.Flush())

			a.Equal([]byte{1, 2, 3, 4, 5, 6}, helper.readAll())
		},
	}.Run()
}
func TestWriteBuffer_Seek(t *testing.T) {
	wbTestHelper{
		t:    t,
		name: "seek",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 5,
				BufferSize:   10,
			}
			// 移動なし
			a.NoError(wb.Seek(0))
			// バッファリングされている範囲内での移動
			a.NoError(wb.Seek(3))
			// バッファサイズを超える位置への移動
			a.NoError(wb.Seek(15))
		},
	}.Run()

	wbTestHelper{
		t:    t,
		name: "",
		fn: func(helper *wbTestHelperArgs) {
			a := helper.a
			wb := WriteBuffer{
				W:            helper.writer,
				MaxWriteSize: 2,
				BufferSize:   5,
			}
			wb.Seek(1)
			a.NoError(wb.Write(copy(wb.WriteBuffer(), []byte{1, 2})))
			a.NoError(wb.Seek(8))
			a.NoError(wb.Write(copy(wb.WriteBuffer(), []byte{8, 9})))
			a.NoError(wb.Flush())

			a.Equal([]byte{0, 1, 2, 0, 0, 0, 0, 0, 8, 9}, helper.readAll())
		},
	}.Run()
}
