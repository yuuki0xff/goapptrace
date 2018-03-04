package storage

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DummyStruct struct {
	Str     string
	Counter int64
}

var DummyData = DummyStruct{
	Str:     "hogehoge",
	Counter: 100,
}

func dummyDataPtr() interface{} {
	return &DummyStruct{}
}

// stripPRW strips private fields.
func stripPRW(v ParallelReadWriter) ParallelReadWriter {
	return ParallelReadWriter{
		File:     v.File,
		ReadOnly: v.ReadOnly,
	}
}

// comparePRWs compares list of ParallelReadWriter.
// This function checks only public fields.
func comparePRWs(a *assert.Assertions, expected, actual []*ParallelReadWriter) {
	exp := make([]ParallelReadWriter, len(expected))
	for i := range expected {
		exp[i] = stripPRW(*expected[i])
	}
	act := make([]ParallelReadWriter, len(actual))
	for i := range actual {
		act[i] = stripPRW(*actual[i])
	}
	a.Equal(exp, act)
}

func simpleFileNamePattern(index int) File {
	index2fpath := func(index int) string {
		return strconv.Itoa(index)
	}
	return File(index2fpath(index))
}

func withTemp(t *testing.T, fn func()) {
	a := assert.New(t)
	tmp, err := ioutil.TempDir("", ".goapptrace.storage.test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	a.NoError(os.Chdir(tmp))
	fn()
}

func TestSplitReadWriter_Open(t *testing.T) {
	// Check error object returned by srw.Open().
	testOpenErr := func(name string, rw SplitReadWriter, err error) {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			a.EqualError(rw.Open(), err.Error())
		})
	}
	// Check srw.file field values after calls srw.Open().
	// This test function checks only public fields on ParallelReadWriter. Private fields
	// are not checked.
	testFiles := func(name string, files []string, prwList []*ParallelReadWriter) {
		t.Run(name, func(t *testing.T) {
			withTemp(t, func() {
				a := assert.New(t)
				// create files
				for _, f := range files {
					_, err := os.Create(f)
					a.NoError(err)
				}

				// open files
				rw := SplitReadWriter{
					FileNamePattern: simpleFileNamePattern,
				}
				a.NoError(rw.Open())

				// check cached files
				comparePRWs(a, prwList, rw.files)
			})
		})
	}

	// FileNamePatternが未指定なら、Open()はエラーを返す。
	testOpenErr("FileNamePattern-is-nil", SplitReadWriter{}, ErrFileNamePatternIsNull)

	// マッチするファイルが無ければ、新しいファイルを1つ作る。
	testFiles("empty", []string{}, []*ParallelReadWriter{
		{File: "0"},
	})

	// 最後のファイル以外は、ReadOnlyにする。
	testFiles("correct", []string{
		"0", "1", "2",
	}, []*ParallelReadWriter{
		{File: "0", ReadOnly: true},
		{File: "1", ReadOnly: true},
		{File: "2"}, // 最後のファイルは書き込み可能
	})
}

func TestSplitReadWriter_Rotate(t *testing.T) {
	withTemp(t, func() {
		a := assert.New(t)
		rw := SplitReadWriter{
			FileNamePattern: simpleFileNamePattern,
		}
		a.NoError(rw.Open())
		comparePRWs(a, []*ParallelReadWriter{
			{File: "0"},
		}, rw.files)

		a.NoError(rw.Rotate())
		comparePRWs(a, []*ParallelReadWriter{
			{File: "0", ReadOnly: true},
			{File: "1"},
		}, rw.files)

		a.NoError(rw.Close())
	})
}

func TestParallelReadWriter_Open(t *testing.T) {
	withTemp(t, func() {
		t.Run("[read-only] cache-is-nil", func(t *testing.T) {
			// read-only modeのときは、cache==nil
			a := assert.New(t)
			rw := ParallelReadWriter{
				File:     File("dummy"),
				ReadOnly: true,
			}
			a.NoError(rw.Open())
			a.Nil(rw.cache)
			a.NoError(rw.Close())
		})
		t.Run("[read-write] cache-and-enc-is-not-nil", func(t *testing.T) {
			a := assert.New(t)
			rw := ParallelReadWriter{
				File: File("dummy"),
			}
			a.NoError(rw.Open())
			a.NotNil(rw.cache)
			a.NotNil(rw.enc)
			a.NoError(rw.Close())
		})
	})
}

func TestParallelReadWriter_Append(t *testing.T) {
	withTemp(t, func() {
		t.Run("read-only", func(t *testing.T) {
			a := assert.New(t)
			rw := ParallelReadWriter{
				File:     File("dummy"),
				ReadOnly: true,
			}
			a.NoError(rw.Open())
			a.EqualError(rw.Append(DummyData), ErrFileIsReadOnly.Error())
			a.NoError(rw.Close())
		})
		t.Run("read-write", func(t *testing.T) {
			a := assert.New(t)
			rw := ParallelReadWriter{
				File: File("dummy"),
			}
			a.NoError(rw.Open())
			a.NoError(rw.Append(DummyData))
			a.NoError(rw.Close())
		})
		t.Run("closed", func(t *testing.T) {
			a := assert.New(t)
			rw := ParallelReadWriter{
				File: File("dummy"),
			}
			a.NoError(rw.Open())
			a.NoError(rw.Close())
			a.EqualError(rw.Append(DummyData), os.ErrClosed.Error())
		})
	})
}

func TestParallelReadWriter_Walk(t *testing.T) {
	testHelper := func(t *testing.T, name string, rw *ParallelReadWriter, prepare func(rw *ParallelReadWriter) error, write func(rw *ParallelReadWriter) error) {
		t.Run(name, func(t *testing.T) {
			withTemp(t, func() {
				a := assert.New(t)
				if prepare != nil {
					a.NoError(prepare(rw))
				}

				a.NoError(rw.Open())
				if write != nil {
					a.NoError(write(rw))
				}
				var i int64
				a.NoError(rw.Walk(dummyDataPtr, func(data interface{}) error {
					a.IsType(&DummyStruct{}, data)

					d := DummyData
					d.Counter = i
					i++
					a.Equal(&d, data)
					return nil
				}))
				a.NoError(rw.Close())
			})
		})
	}

	writeRecords := func(n, first int) func(rw *ParallelReadWriter) error {
		return func(rw *ParallelReadWriter) error {
			for i := first; i < n; i++ {
				d := &DummyStruct{}
				*d = DummyData
				d.Counter = int64(i)
				if err := rw.Append(d); err != nil {
					return err
				}
			}
			return nil
		}
	}

	prepareEmpty := func(ro *ParallelReadWriter) error {
		rw := *ro
		rw.ReadOnly = false
		if err := rw.Open(); err != nil {
			return err
		}
		return rw.Close()
	}
	prepareRecords := func(n int) func(ro *ParallelReadWriter) error {
		return func(ro *ParallelReadWriter) error {
			rw := *ro
			rw.ReadOnly = false
			if err := rw.Open(); err != nil {
				return err
			}
			writeRecords(n, 0)(&rw)
			return rw.Close()
		}
	}

	t.Run("read-only/no-cache", func(t *testing.T) {
		// 空のファイル(レコード数が0)をWalk()できる
		testHelper(t, "empty", &ParallelReadWriter{
			File:     File("dummy"),
			ReadOnly: true,
		}, prepareEmpty, nil)

		// いくつかのレコードがあるファイルをWalk()できる
		testHelper(t, "non-empty", &ParallelReadWriter{
			File:     File("dummy"),
			ReadOnly: true,
		}, prepareRecords(100), writeRecords(50, 100))
	})
	t.Run("read-write", func(t *testing.T) {
		// read-writeでは、キャッシュを必ず使用しなければならない。

		t.Run("empty", func(t *testing.T) {
			// 空のファイルをWalk()できる
			testHelper(t, "no-append", &ParallelReadWriter{
				File: File("dummy"),
			}, prepareEmpty, nil)

			// 空のファイルに追記してもWalk()できる
			testHelper(t, "append-100-records", &ParallelReadWriter{
				File: File("dummy"),
			}, prepareEmpty, writeRecords(100, 0))
		})
		t.Run("non-empty", func(t *testing.T) {
			// いくつかのレコードがあるファイルをWalk()できる
			testHelper(t, "no-write", &ParallelReadWriter{
				File: File("dummy"),
			}, prepareRecords(100), nil)

			// 追記した後でもWalk()できる
			testHelper(t, "append-50-records", &ParallelReadWriter{
				File: File("dummy"),
			}, prepareRecords(100), writeRecords(50, 100))
		})
	})
}
