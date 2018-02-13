package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func TestLogID_Hex(t *testing.T) {
	logID := LogID{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if len(logID[:]) != 16 {
		t.Fatal("LogID length is not 16byte")
	}
	if logID.Hex() != "0f0e0d0c0b0a09080706050403020100" {
		t.Fatal("LogID.Hex() returns wrong hex id")
	}
}

func TestLogID_Unhex(t *testing.T) {
	a := assert.New(t)
	logID := LogID{}

	if _, err := logID.Unhex(""); err == nil {
		t.Fatal("LogID.Unhex() should raise error for non-16byte id")
	}
	if _, err := logID.Unhex("0"); err == nil {
		t.Fatal("LogID.Unhex() should raise error for invalid hex string")
	}
	{
		id, err := logID.Unhex("000102030405060708090a0b0c0d0e0f")
		if err != nil {
			t.Fatal("LogID.Unhex() should not raise error for valid id")
		}
		if !bytes.Equal(id[:], []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}) {
			t.Fatal("LogID.Unhex() returns wrong id")
		}
	}
}

func TestLog_withEmptyFile(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer func() {
		if err = os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:       LogID{},
		Root:     dirlayout,
		Metadata: &LogMetadata{},
	}
	a.NoError(l.Open())
	a.NoError(err)
	a.NoError(l.Close())

	if l.Symbols() == nil {
		a.NoError(errors.New("should returns not nil, but got nil"))
	}
	a.NoError(l.WalkRawFuncLog(func(evt logutil.RawFuncLog) error {
		return errors.New("should not contains any log record, but found a log record")
	}), "Log.WalkRawFuncLog():")
	a.NoError(l.Close())
}

func TestLog_AppendRawFuncLog(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer func() {
		if err = os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &LogMetadata{},
		MaxFileSize: 1,
		// 自動ローテーションを発生させるため
		rotateInterval: 1,
	}
	a.NoError(l.Open())
	a.NoError(l.AppendRawFuncLog(&logutil.RawFuncLog{}))
	a.NoError(l.AppendRawFuncLog(&logutil.RawFuncLog{}))

	// data dir should only contains those files:
	//   xxxx.0.func.log.gz
	//   xxxx.0.rawfunc.log.gz
	//   xxxx.0.goroutine.log.gz
	//   xxxx.1.func.log.gz
	//   xxxx.1.rawfunc.log.gz
	//   xxxx.1.goroutine.log.gz
	//   xxxx.index.gz
	//   xxxx.symbol.gz
	files, err := ioutil.ReadDir(dirlayout.DataDir())
	if err != nil {
		panic(err)
	}
	for i := range files {
		t.Logf("files[%d] = %s", i, files[i].Name())
	}
	if len(files) != 8 {
		t.Fatalf("data file count: (god) %d != %d (expected)", len(files), 6)
	}

	var i int
	a.NoError(l.WalkRawFuncLog(func(evt logutil.RawFuncLog) error {
		i++
		return nil
	}), "Log.WalkRawFuncLog():")
	if i != 2 {
		a.NoError(fmt.Errorf("log records: (got) %d != %d (expected)", i, 2))
	}

	a.NoError(l.Close())
}

// Logで書き込みながら、Logで正しく読み込めるかテスト。
func TestLog_ReadDuringWriting(t *testing.T) {
	a := assert.New(t)
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	a.NoError(err)
	defer func() {
		if err := os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	a.NoError(dirlayout.Init())

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &LogMetadata{},
		MaxFileSize: 1000,
		// 自動ローテーションを発生させるため
		rotateInterval: 10,
	}
	a.NoError(l.Open())

	checkRecordCount := func(expect int64) error {
		var actual int64
		l.WalkRawFuncLog(func(evt logutil.RawFuncLog) error {
			actual++
			return nil
		})
		if actual != expect {
			return fmt.Errorf("mismatch log record count: expect=%d actual=%d", expect, actual)
		}
		return nil
	}

	// ファイルのローテーションが2回発生するまでレコードを追加する。
	// ローテーションをしてもRawFuncLogCacheが正しくクリアできるかテストする。
	for i := int64(0); l.index.Len() < 3; i++ {
		a.NoError(checkRecordCount(i))
		// 書き込み先のファイルはgzip圧縮されている。
		// 同じデータが連続していると大幅に圧縮されてしまい、いつまで経ってもファイルのローテーションが発生しない。
		// このような自体を回避するために、乱数を使用して圧縮率を低くする。
		randomName := string([]rune{
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
			rune(rand.Int()),
		})
		a.NoError(l.AppendRawFuncLog(&logutil.RawFuncLog{
			ID:   logutil.RawFuncLogID(i),
			Tag:  logutil.TagName(randomName),
			GID:  logutil.GID(rand.Int()),
			TxID: logutil.TxID(rand.Int()),
		}), "Log.AppendRawFuncLog():")

		// RawFuncLogが1つあたり0.1バイト未満で書き込まれるのは考えにくい。
		// 十分な回数だけ試行しても終了しない場合、テスト失敗として扱う。
		if i > l.MaxFileSize*30 {
			t.Fatal("loop count limit reached")
		}
	}
	a.NoError(l.Close())
}
