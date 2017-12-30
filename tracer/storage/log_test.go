package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

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
		if bytes.Compare(id[:], []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}) != 0 {
			t.Fatal("LogID.Unhex() returns wrong id")
		}
	}
}

func TestLog_withEmptyFile(t *testing.T) {
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	must(t, err, "can not create a temporary directory:")
	defer func() {
		if err = os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	must(t, dirlayout.Init(), "DirLayout.Init():")

	l := Log{
		ID:       LogID{},
		Root:     dirlayout,
		Metadata: &LogMetadata{},
	}
	must(t, l.Open(), "Log.Open():")
	must(t, err, "Log.Writer():")
	must(t, l.Close(), "Log.Close():")

	if l.Symbols() == nil {
		must(t, errors.New("should returns not nil, but got nil"), "Log.Symbols():")
	}
	must(t, l.Walk(func(evt logutil.RawFuncLog) error {
		return errors.New("should not contains any log record, but found a log record")
	}), "Log.Walk():")
	must(t, l.Close(), "Log.Close():")
}

func TestLog_AppendRawFuncLog(t *testing.T) {
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	must(t, err, "can not create a temporary directory:")
	defer func() {
		if err = os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	must(t, dirlayout.Init(), "DirLayout.Init():")

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &LogMetadata{},
		MaxFileSize: 1,
	}
	must(t, l.Open(), "Log.Open():")
	must(t, l.AppendRawFuncLog(&logutil.RawFuncLog{}), "Log.AppendRawFuncLog():")
	must(t, l.AppendRawFuncLog(&logutil.RawFuncLog{}), "Log.AppendRawFuncLog():")
	must(t, l.Close(), "Log.Close():")

	// data dir should only contains those files:
	//   xxxx.0.func.log.gz
	//   xxxx.0.rawfunc.log.gz
	//   xxxx.1.func.log.gz
	//   xxxx.1.rawfunc.log.gz
	//   xxxx.index.gz
	//   xxxx.symbol.gz
	files, err := ioutil.ReadDir(dirlayout.DataDir())
	if err != nil {
		panic(err)
	}
	for i := range files {
		t.Logf("files[%d] = %s", i, files[i].Name())
	}
	if len(files) != 6 {
		t.Fatalf("data file count: (god) %d != %d (expected)", len(files), 6)
	}

	var i int
	must(t, l.Walk(func(evt logutil.RawFuncLog) error {
		i++
		return nil
	}), "Log.Walk():")
	if i != 2 {
		must(t, fmt.Errorf("log records: (got) %d != %d (expected)", i, 2), "Log.Walk():")
	}
}

// Logで書き込みながら、Logで正しく読み込めるかテスト。
func TestLog_ReadDuringWriting(t *testing.T) {
	tempdir, err := ioutil.TempDir("", ".goapptrace_storage")
	must(t, err, "can not create a temporary directory:")
	defer func() {
		if err := os.RemoveAll(tempdir); err != nil {
			panic(err)
		}
	}()
	dirlayout := DirLayout{Root: tempdir}
	must(t, dirlayout.Init(), "DirLayout.Init():")

	l := Log{
		ID:          LogID{},
		Root:        dirlayout,
		Metadata:    &LogMetadata{},
		MaxFileSize: 1000,
	}
	must(t, l.Open(), "Log.Open():")

	checkRecordCount := func(expect int64) error {
		var actual int64
		l.Walk(func(evt logutil.RawFuncLog) error {
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
		must(t, checkRecordCount(i), "checkRecordCount(0):")
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
		must(t, l.AppendRawFuncLog(&logutil.RawFuncLog{
			Time: logutil.Time(i),
			Tag:  logutil.TagName(randomName),
			GID:  logutil.GID(rand.Int()),
			TxID: logutil.TxID(rand.Int()),
		}), "Log.AppendRawFuncLog():")

		// RawFuncLogNewが1つあたり0.1バイト未満で書き込まれるのは考えにくい。
		// 十分な回数だけ試行しても終了しない場合、テスト失敗として扱う。
		if i > l.MaxFileSize*30 {
			t.Fatal("loop count limit reached")
		}
	}
	must(t, l.Close(), "Log.Close():")
}
