package storage

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func TestLogID_Hex(t *testing.T) {
	a := assert.New(t)
	logID := LogID{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}

	a.Len(logID[:], 16)
	a.Equal("0f0e0d0c0b0a09080706050403020100", logID.Hex())
}

func TestLogID_Unhex(t *testing.T) {
	a := assert.New(t)
	logID := LogID{}

	var err error
	_, err = logID.Unhex("")
	a.Error(err, "LogID.Unhex() should raise error for non-16byte id")
	_, err = logID.Unhex("0")
	a.Error(err, "LogID.Unhex() should raise error for invalid hex string")

	id, err := logID.Unhex("000102030405060708090a0b0c0d0e0f")
	a.NoError(err, "LogID.Unhex() should not raise error for valid id")
	a.Equal([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, id[:])
}

func TestLog_withEmptyFile(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer a.NoError(os.RemoveAll(tempdir))
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:       LogID{},
		Root:     dirlayout,
		Metadata: &types.LogMetadata{},
	}
	a.NoError(l.Open())
	a.NoError(l.Close())

	a.NotNil(l.Symbols())
	var called bool
	l.RawFuncLog(func(store *RawFuncLogStore) {
		a.Equal(int64(0), store.Records(), "should not contains any log record, but found a log record")
		called = true
	})
	a.True(called)
	a.NoError(l.Close())
}

func TestLog_AppendRawFuncLog(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer a.NoError(os.RemoveAll(tempdir))
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &types.LogMetadata{},
		MaxFileSize: 1,
	}
	a.NoError(l.Open())
	l.RawFuncLog(func(store *RawFuncLogStore) {
		a.NoError(store.SetNolock(&types.RawFuncLog{ID: 0}))
		a.NoError(store.SetNolock(&types.RawFuncLog{ID: 1}))
	})

	var i int
	l.RawFuncLog(func(store *RawFuncLogStore) {
		i = int(store.Records())
	})
	a.Equal(2, i)

	a.NoError(l.Close())

	// data dir should only contains those files:
	//   xxxx.0.func.log
	//   xxxx.0.rawfunc.log
	//   xxxx.0.goroutine.log
	//   xxxx.index
	//   xxxx.symbol
	files, err := ioutil.ReadDir(dirlayout.DataDir())
	a.NoError(err)
	for i := range files {
		t.Logf("files[%d] = %s", i, files[i].Name())
	}
	a.Len(files, 5)
}

// Logで書き込みながら、Logで正しく読み込めるかテスト。
func TestLog_ReadDuringWriting(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer a.NoError(os.RemoveAll(tempdir))
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &types.LogMetadata{},
		MaxFileSize: 1000,
	}
	a.NoError(l.Open())

	l.RawFuncLog(func(store *RawFuncLogStore) {
		for i := int64(0); i < 1000; i++ {
			// レコード数が一致しているか
			a.Equal(i, store.Records())

			// 新しいレコードを追加
			a.NoError(store.SetNolock(&types.RawFuncLog{
				ID:   types.RawFuncLogID(i),
				Tag:  types.TagName(i),
				GID:  types.GID(i),
				TxID: types.TxID(i),
			}), "Log.AppendRawFuncLog():")

			// 書き込み済みのレコードが読み出せるか
			var raw types.RawFuncLog
			// 書き込み済みのレコードが読み出せるか
			randIdx := rand.Int63n(i + 1)
			a.NoError(store.GetNolock(types.RawFuncLogID(randIdx), &raw))
			a.Equal(types.RawFuncLog{
				ID:   types.RawFuncLogID(randIdx),
				Tag:  types.TagName(randIdx),
				GID:  types.GID(randIdx),
				TxID: types.TxID(randIdx),
			}, raw)
		}
	})
	a.NoError(l.Close())
}
