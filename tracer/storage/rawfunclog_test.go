package storage

import (
	"os"
	"testing"

	"bytes"
	"encoding/json"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func newRawFuncLogNew(startTime, endTime int64, frames []logutil.FuncStatusID) (start, end *logutil.RawFuncLogNew) {
	start = &logutil.RawFuncLogNew{
		Time:      logutil.Time(startTime),
		Tag:       "funcStart",
		Timestamp: startTime,
		Frames:    frames,
		TxID:      logutil.NewTxID(),
	}
	end = &logutil.RawFuncLogNew{
		Time:      logutil.Time(endTime),
		Tag:       "funcEnd",
		Timestamp: endTime,
		Frames:    frames,
		TxID:      start.TxID,
	}
	return
}

func doTestRawFunclogReaderWriter(
	t *testing.T,
	writeFunc func(writer *RawFuncLogWriter),
	checkFunc func(funclog logutil.RawFuncLogNew) error,
) {
	file := createTempFile()
	defer must(t, os.Remove(string(file)), "Delete tmpfile:")

	// write phase
	{
		fw := RawFuncLogWriter{
			File: file,
		}
		must(t, fw.Open(), "RawFuncLogWriter.Open():")
		writeFunc(&fw)
		must(t, fw.Close(), "RawFuncLogWriter.Close():")
	}

	// read phase
	{
		fr := RawFuncLogReader{
			File: file,
		}
		must(t, fr.Open(), "RawFuncLogReader.Open():")
		fr.Walk(checkFunc)
		must(t, fr.Close(), "RawFuncLogReader.Close():")
	}
}

func TestRawFunclogReaderWriter_emptyFile(t *testing.T) {
	doTestRawFunclogReaderWriter(
		t,
		func(writer *RawFuncLogWriter) {},
		func(funclog logutil.RawFuncLogNew) error {
			t.Fatalf("This func should not call, but called with funclog=%+v", funclog)
			return nil
		},
	)
}

func TestRawFunclogReaderWriter_data(t *testing.T) {
	frames1 := []logutil.FuncStatusID{0}
	frames2 := []logutil.FuncStatusID{0, 1}
	frames3 := []logutil.FuncStatusID{0, 1, 2}
	frames4 := []logutil.FuncStatusID{0, 1, 2, 2}
	frames5 := []logutil.FuncStatusID{0, 1, 2, 3}
	flog0, flog9 := newRawFuncLogNew(0, 9, frames1)
	flog1, flog8 := newRawFuncLogNew(1, 8, frames2)
	flog2, flog7 := newRawFuncLogNew(2, 7, frames3)
	flog3, flog4 := newRawFuncLogNew(3, 4, frames4)
	flog5, flog6 := newRawFuncLogNew(5, 6, frames5)

	funcLogs := []*logutil.RawFuncLogNew{
		flog0,
		flog1,
		flog2,
		flog3,
		flog4,
		flog5,
		flog6,
		flog7,
		flog8,
		flog9,
	}

	var nextReadIdx int
	doTestRawFunclogReaderWriter(
		t,
		func(writer *RawFuncLogWriter) {
			for i := range funcLogs {
				must(t, writer.Append(funcLogs[i]), "writer.Append():")
			}
		},
		func(funclog logutil.RawFuncLogNew) error {
			expectedFuncLog, err := json.Marshal(funcLogs[nextReadIdx])
			must(t, err, "json.Marshall():")
			readedFuncLog, err := json.Marshal(funclog)
			must(t, err, "json.Marshall():")

			t.Logf("funcLogs[%d] = %s", nextReadIdx, string(expectedFuncLog))

			if bytes.Compare(expectedFuncLog, readedFuncLog) != 0 {
				t.Errorf("expected RawFuncLogNew is %s, but %s", string(expectedFuncLog), string(readedFuncLog))
			}
			nextReadIdx++
			return nil
		},
	)
	if nextReadIdx != 10 {
		t.Fatalf("record count is mismatched: expected to be %d, but %d", 10, nextReadIdx)
	}
}
