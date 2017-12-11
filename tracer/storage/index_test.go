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
			Timestamp: time.Unix(10, 0),
			Records:   10,
		}, {
			Timestamp: time.Unix(30, 0),
			Records:   20,
		}, {
			Timestamp: time.Unix(50, 0),
			Records:   15,
		}, {
			Timestamp: time.Unix(90, 0),
			Records:   5,
		}, {
			Timestamp: time.Unix(110, 0),
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
	must(t, index.Open(), "Index.Open():")
	for i := range records {
		must(t, index.Append(records[i]), "Index.Append():")
	}
	must(t, index.Close(), "Index.Close():")

	must(t, index2.Open(), "Index.Open():")
	must(t, index2.Load(), "index.Load():")
	var i int
	if err := index3.Walk(func(i int64, ir IndexRecord) error {
		i++
		return nil
	}); err != nil {
		t.Fatalf("Index.Walk() should not return any error, but %+v", err)
	}
	if i != len(records) {
		t.Fatalf("mismatch record count: expect=%d actual=%d", len(records), i)
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
	must(t, index2.Close(), "Index.Close():")

	must(t, index3.Open(), "Index.Open():")
	must(t, index3.Load(), "Index.Load():")
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
	must(t, index3.Close(), "Index.Close():")
}
