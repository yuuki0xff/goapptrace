package storage

import (
	"os"
	"testing"
	"time"
)

func TestIndex(t *testing.T) {
	file := createTempFile()
	defer os.Remove(string(file))

	records := []IndexRecord{
		{
			Timestamps: time.Unix(10, 0),
			Records:    10,
		}, {
			Timestamps: time.Unix(30, 0),
			Records:    20,
		}, {
			Timestamps: time.Unix(50, 0),
			Records:    15,
		}, {
			Timestamps: time.Unix(90, 0),
			Records:    5,
		}, {
			Timestamps: time.Unix(110, 0),
			Records:    50,
		},
	}

	index := Index{
		File: file,
	}
	must(t, index.Open(), "Index.Open():")
	for i := range records {
		must(t, index.Append(records[i]), "Index.Append():")
	}
	must(t, index.Close(), "Index.Close():")

	must(t, index.Open(), "Index.Open():")
	must(t, index.Load(), "index.Load():")
	must(t, index.Close(), "Index.Close():")

	if len(records) != len(index.records) {
		t.Errorf("records count is mismatched: expect %d, but %d", len(records), len(index.records))
	}
	for i := range records {
		t.Logf("Index.records[%d] = %+v", i, index.records[i])

		if records[i].Timestamps != index.records[i].Timestamps {
			t.Errorf("Timestamps is not matched: expect %d, but %d", records[i].Timestamps, index.records[i].Timestamps)
		}
		if records[i].Records != index.records[i].Records {
			t.Errorf("Records is not matched: expect %d, but %d", records[i].Records, index.records[i].Records)
		}
	}

	must(t, index.Open(), "Index.Open():")
	err := index.Walk(func(i int64, ir IndexRecord) error {
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
	must(t, index.Close(), "Index.Close():")
}
