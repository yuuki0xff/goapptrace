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
	a.NoError(index2.Walk(func(i int64, ir IndexRecord) error {
		recordCount++
		return nil
	}))
	a.Len(records, recordCount)
	a.Len(index2.records, recordCount)

	for i := range records {
		t.Logf("Index.records[%d] = %+v", i, index2.records[i])

		a.Equal(records[i].Timestamp, index.records[i].Timestamp)
		a.Equal(records[i].Records, index.records[i].Records)
	}
	a.NoError(index2.Close())

	a.NoError(index3.Open())
	a.NoError(index3.Load())
	err := index3.Walk(func(i int64, ir IndexRecord) error {
		t.Logf("Index.Walk(): i=%d, record=%+v", i, ir)
		if i == 2 {
			return StopIteration
		}
		a.False(i > 2, "Index.Walk() should break the loop when i==2, but it seems to be continuing")
		return nil
	})
	a.Equal(StopIteration, err, "Index.Walk() should return StopIteration")
	a.NoError(index3.Close())
}
