package storage

import (
	"io/ioutil"
	"os"
	"testing"

	set "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
)

func setupStorage() (storage *Storage, cleanup func()) {
	tmpdir, err := ioutil.TempDir("", ".goapptrace_storage")
	if err != nil {
		panic(err)
	}

	storage = &Storage{
		Root: DirLayout{
			Root: tmpdir,
		},
	}
	cleanup = func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			panic(err)
		}
	}
	return
}

func setupStorageDir(t *testing.T) (dir DirLayout, logIDSet set.Set, cleanup func()) {
	a := assert.New(t)
	var strg *Storage
	strg, cleanup = setupStorage()
	dir = strg.Root

	a.NoError(strg.Init())

	logIDSet = set.NewSet()
	for i := 0; i < 2; i++ {
		logobj, err := strg.New()
		a.NoError(err)
		a.NotNil(logobj)
		logIDSet.Add(logobj.ID)
	}

	a.NoError(strg.Close())
	return
}

func logIDSetFromLogs(logs []*Log) (logIDSet set.Set) {
	logIDSet = set.NewSet()
	for _, logobj := range logs {
		logIDSet.Add(logobj.ID)
	}
	return
}

func TestStorage(t *testing.T) {
	a := assert.New(t)
	strg, cleanup := setupStorage()
	defer cleanup()

	a.NoError(strg.Init())

	logs, err := strg.Logs()
	a.NoError(err)
	a.Len(logs, 0)

	logobj, err := strg.New()
	a.NoError(err)
	a.NotNil(logobj)

	logs, err = strg.Logs()
	a.NoError(err)
	a.Len(logs, 1)

	logobj2, ok := strg.Log(logobj.ID)
	a.Truef(ok, "Storage.Log(): not found %s", logobj.ID.Hex())
	a.Equal(logobj, logobj2)
	a.NoError(strg.Close())
}

func TestStorage_Load(t *testing.T) {
	a := assert.New(t)
	// prepare directory
	dir, logIDSet, cleanup := setupStorageDir(t)
	defer cleanup()

	strg := Storage{
		Root: dir,
	}
	// load logs from files
	a.NoError(strg.Init())

	newLogs, err := strg.Logs()
	a.NoError(err)
	newLogIDSet := logIDSetFromLogs(newLogs)

	a.Truef(logIDSet.Equal(newLogIDSet), "Missmatch logs: expect %+v, but %+v", logIDSet, newLogIDSet)
	a.NoError(strg.Close())
}
