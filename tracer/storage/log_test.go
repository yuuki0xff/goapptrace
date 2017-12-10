package storage

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"errors"

	"fmt"

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
		if err := os.RemoveAll(tempdir); err != nil {
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
	must(t, l.Init(), "Log.Init():")
	lw, err := l.Writer()
	must(t, err, "Log.Writer():")
	must(t, lw.Close(), "LogWriter.Close():")

	lr, err := l.Reader()
	must(t, err, "Log.Reader():")
	if lr.Symbols() == nil {
		must(t, errors.New("should returns not nil, but got nil"), "LogReader.Symbols():")
	}
	must(t, lr.Walk(func(evt logutil.RawFuncLogNew) error {
		return errors.New("should not contains any log record, but found a log record")
	}), "LogReader.Walk():")
	must(t, lr.Close(), "LogReader.Close():")
}

func TestLog_AppendFuncLog(t *testing.T) {
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
		MaxFileSize: 1,
	}
	must(t, l.Init(), "Log.Init():")
	lw, err := l.Writer()
	must(t, err, "Log.Writer():")
	must(t, lw.AppendFuncLog(&logutil.RawFuncLogNew{}), "LogWriter.AppendFuncLog():")
	must(t, lw.AppendFuncLog(&logutil.RawFuncLogNew{}), "LogWriter.AppendFuncLog():")
	must(t, lw.Close(), "LogWriter.Close():")

	// data dir should only contains those files:
	//   xxxx.0.rawfunc.log.gz
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
	if len(files) != 4 {
		t.Fatal("data file count is mismatched")
	}

	lr, err := l.Reader()
	must(t, err, "Log.Reader():")
	var i int
	must(t, lr.Walk(func(evt logutil.RawFuncLogNew) error {
		i++
		return nil
	}), "LogReader.Walk():")
	if i != 2 {
		must(t, fmt.Errorf("log records: (got) %d != %d (expected)", i, 2), "LogReader.Walk():")
	}

}
