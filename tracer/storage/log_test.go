package storage

import (
	"errors"
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
	a.NoError(err)
	a.NoError(l.Close())

	a.NotNil(l.Symbols())
	a.NoError(l.WalkRawFuncLog(func(evt types.RawFuncLog) error {
		return errors.New("should not contains any log record, but found a log record")
	}), "Log.WalkRawFuncLog():")
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
		// 自動ローテーションを発生させるため
		rotateInterval: 1,
	}
	a.NoError(l.Open())
	a.NoError(l.AppendRawFuncLog(&types.RawFuncLog{}))
	a.NoError(l.AppendRawFuncLog(&types.RawFuncLog{}))

	var i int
	a.NoError(l.WalkRawFuncLog(func(evt types.RawFuncLog) error {
		i++
		return nil
	}), "Log.WalkRawFuncLog():")
	a.Equal(2, i)

	a.NoError(l.Close())

	// data dir should only contains those files:
	//   xxxx.0.func.log
	//   xxxx.0.rawfunc.log
	//   xxxx.0.goroutine.log
	//   xxxx.1.func.log
	//   xxxx.1.rawfunc.log
	//   xxxx.1.goroutine.log
	//   xxxx.index
	//   xxxx.symbol
	files, err := ioutil.ReadDir(dirlayout.DataDir())
	a.NoError(err)
	for i := range files {
		t.Logf("files[%d] = %s", i, files[i].Name())
	}
	a.Len(files, 8)
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
		// 自動ローテーションを発生させるため
		rotateInterval: 10,
	}
	a.NoError(l.Open())

	checkRecordCount := func(expect int64) error {
		var actual int64
		l.WalkRawFuncLog(func(evt types.RawFuncLog) error {
			actual++
			return nil
		})
		a.Equal(expect, actual)
		return nil
	}

	// ファイルのローテーションが2回発生するまでレコードを追加する。
	// ローテーションをしてもRawFuncLogCacheが正しくクリアできるかテストする。
	for i := int64(0); l.index.Len() < 3; i++ {
		a.NoError(checkRecordCount(i))
		// 書き込み先のファイルは圧縮されていた場合、同じデータが連続していると大幅に圧縮されてしまう。
		// そのため、いつまで経ってもファイルのローテーションが発生しない可能性がある。
		// このような問題を回避するために、乱数を使用して圧縮率を低くする。
		a.NoError(l.AppendRawFuncLog(&types.RawFuncLog{
			ID:   types.RawFuncLogID(i),
			Tag:  types.TagName(rand.Uint32()),
			GID:  types.GID(rand.Int()),
			TxID: types.TxID(rand.Int()),
		}), "Log.AppendRawFuncLog():")

		// RawFuncLogが1つあたり0.1バイト未満で書き込まれるのは考えにくい。
		// 十分な回数だけ試行しても終了しない場合、テスト失敗として扱う。
		if i > l.MaxFileSize*30 {
			t.Fatal("loop count limit reached")
		}
	}
	a.NoError(l.Close())
}
