package storage

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func TestIndex(t *testing.T) {
	a := assert.New(t)
	file := createTempFile()
	defer os.Remove(string(file))

	records := []IndexRecord{
		{
			Timestamp: logutil.NewTime(time.Unix(10, 0)),
			Records:   10,
		}, {
			Timestamp: logutil.NewTime(time.Unix(30, 0)),
			Records:   20,
		}, {
			Timestamp: logutil.NewTime(time.Unix(50, 0)),
			Records:   15,
		}, {
			Timestamp: logutil.NewTime(time.Unix(90, 0)),
			Records:   5,
		}, {
			Timestamp: logutil.NewTime(time.Unix(110, 0)),
			Records:   50,
		},
	}

	index := Index{
		File: file,
	}
	index2 := Index{
		File: file,
	}
	index3 := Index{
		File: file,
	}
	a.NoError(index.Open())
	for i := range records {
		a.NoError(index.Append(records[i]))
	}
	a.NoError(index.Close())

	a.NoError(index2.Open())
	a.NoError(index2.Load())
	var recordCount int
	if err := index2.Walk(func(i int64, ir IndexRecord) error {
		recordCount++
		return nil
	}); err != nil {
		t.Fatalf("Index.Walk() should not return any error, but %+v", err)
	}
	if recordCount != len(records) {
		t.Fatalf("mismatch record count: expect=%d actual=%d", len(records), recordCount)
	}
	if len(records) != len(index2.records) {
		t.Errorf("records count is mismatched: expect %d, but %d", len(records), len(index.records))
	}
	for i := range records {
		t.Logf("Index.records[%d] = %+v", i, index2.records[i])

		if records[i].Timestamp != index2.records[i].Timestamp {
			t.Errorf("Timestamp is not matched: expect %d, but %d", records[i].Timestamp, index.records[i].Timestamp)
		}
		if records[i].Records != index2.records[i].Records {
			t.Errorf("Records is not matched: expect %d, but %d", records[i].Records, index.records[i].Records)
		}
	}
	a.NoError(index2.Close())

	a.NoError(index3.Open())
	a.NoError(index3.Load())
	err := index3.Walk(func(i int64, ir IndexRecord) error {
		t.Logf("Index.Walk(): i=%d, record=%+v", i, ir)
		if i == 2 {
			return StopIteration
		}
		if i > 2 {
			t.Fatalf("Index.Walk() should break the loop when i==2, but it seems to be continuing")
		}
		return nil
	})
	if err != StopIteration {
		t.Errorf("Index.Walk() should return StopIteration, but %+v", err)
	}
	a.NoError(index3.Close())
}
