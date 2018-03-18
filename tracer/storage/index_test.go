package storage

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

var indexRecords = []IndexRecord{
	{MinID: 1, MaxID: 5, MinStart: 1000, MaxStart: 1040, MinEnd: 1010, MaxEnd: 1050},
	{MinID: 6, MaxID: 9, MinStart: 1050, MaxStart: 1090, MinEnd: 1060, MaxEnd: 1100},
	{MinID: 10, MaxID: 19, MinStart: 1100, MaxStart: 1200, MinEnd: 1150, MaxEnd: 1900},
	{MinID: 20, MaxID: 29, MinStart: 1201, MaxStart: 1299, MinEnd: 1230, MaxEnd: 1700},
	{MinID: 30, MaxID: 39, MinStart: 1300, MaxStart: 1400, MinEnd: 1410, MaxEnd: 1500},
}
var indexBytes []byte

func init() {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(indexRecords)
	if err != nil {
		panic(err)
	}
	indexBytes = b.Bytes()
}

func TestIndex_Open(t *testing.T) {
	a := assert.New(t)
	a.NotPanics(func() {
		idx := Index{}
		idx.Open()
	})
}
func TestIndex_Load(t *testing.T) {
	a := assert.New(t)
	util.WithTempFile(func(tmpfile string) {
		a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
		idx := Index{
			File: File(tmpfile),
		}
		a.NoError(idx.Load())
	})
}
func TestIndex_Save(t *testing.T) {
	t.Run("read-only", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
			idx := Index{
				File:     File(tmpfile),
				ReadOnly: true,
			}
			a.EqualError(idx.Save(), ErrReadOnly.Error())
		})
	})

	t.Run("writable", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			idx := Index{
				File: File(tmpfile),
			}
			for _, rec := range indexRecords {
				a.NoError(idx.Append(rec))
			}
			a.NoError(idx.Save())

			size, err := idx.File.Size()
			a.NoError(err)
			a.Equal(int64(len(indexBytes)), size)
		})
	})
}
func TestIndex_Append(t *testing.T) {
	t.Run("read-only", func(t *testing.T) {
		a := assert.New(t)
		idx := Index{
			ReadOnly: true,
		}
		a.EqualError(idx.Append(IndexRecord{}), ErrReadOnly.Error())
	})

	t.Run("auto-load", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
			idx := Index{
				File: File(tmpfile),
			}
			for _, rec := range indexRecords {
				a.NoError(idx.Append(rec))
			}
			a.Len(idx.records, 2*len(indexRecords))
		})
	})
}
func TestIndex_Close(t *testing.T) {
	t.Run("read-only", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
			idx := Index{
				File:     File(tmpfile),
				ReadOnly: true,
			}
			a.NoError(idx.Close())
			size, err := idx.File.Size()
			a.NoError(err)
			a.Equal(int64(len(indexBytes)), size)
		})
	})

	t.Run("save", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
			idx := Index{
				File: File(tmpfile),
			}
			for _, rec := range indexRecords {
				a.NoError(idx.Append(rec))
			}
			a.NoError(idx.Close())

			size, err := idx.File.Size()
			a.NoError(err)
			a.True(int64(len(indexBytes)) < size)
		})
	})
}
func TestIndex_IDRangeByTime(t *testing.T) {
	t.Run("not-found", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
			idx := Index{
				File:     File(tmpfile),
				ReadOnly: true,
			}
			s, e := idx.IDRangeByTime(0, 0)
			a.Equal(int64(0), s)
			a.Equal(int64(0), e)
		})
	})

	helper := func(name string, start, end types.Time, startId, endId int64) {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			util.WithTempFile(func(tmpfile string) {
				a.NoError(ioutil.WriteFile(tmpfile, indexBytes, 0600))
				idx := Index{
					File:     File(tmpfile),
					ReadOnly: true,
				}
				s, e := idx.IDRangeByTime(start, end)
				a.Equal(startId, s)
				a.Equal(endId, e)
			})
		})
	}

	helper("contain", 1020, 1080, 1, 9)
	helper("overlap", 1120, 1280, 10, 29)
	helper("overlap", 1120, 1280, 10, 29)
}
