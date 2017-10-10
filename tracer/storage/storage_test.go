package storage

import (
	"io/ioutil"
	"os"
	"testing"

	set "github.com/deckarep/golang-set"
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
	var strg *Storage
	strg, cleanup = setupStorage()
	dir = strg.Root

	must(t, strg.Init(), "Storage.Init():")

	logIDSet = set.NewSet()
	for i := 0; i < 2; i++ {
		logobj, err := strg.New()
		must(t, err, "Storage.New():")
		if logobj == nil {
			t.Fatalf("Storage.New() should not return nil")
		}
		must(t, logobj.Close(), "lobobj.Close():")
		logIDSet.Add(logobj.ID)
	}
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
	strg, cleanup := setupStorage()
	defer cleanup()

	must(t, strg.Init(), "Storage.Init():")

	logs, err := strg.Logs()
	must(t, err, "Storage.Logs():")
	if len(logs) != 0 {
		t.Fatalf("Storage.Logs() should return empty array, but %+v", logs)
	}

	logobj, err := strg.New()
	must(t, err, "Storage.New():")
	if logobj == nil {
		t.Fatalf("Storage.New() should not return nil")
	}

	logs, err = strg.Logs()
	must(t, err, "Storage.Logs():")
	if len(logs) != 1 {
		t.Fatalf("Storage.Logs() returns wrong result: I expected a log object, but %+v", logs)
	}

	logobj2, ok := strg.Log(logobj.ID)
	if !ok {
		t.Fatalf("Storage.Log(): not found %s", logobj.ID.Hex())
	}
	if logobj != logobj2 {
		t.Fatalf("Storage.Log(): returns different object: obj1=%+v obj2=%+v", logobj, logobj2)
	}
}

func TestStorage_Load(t *testing.T) {
	// prepare directory
	dir, logIDSet, cleanup := setupStorageDir(t)
	defer cleanup()

	strg := Storage{
		Root: dir,
	}
	// load logs from files
	must(t, strg.Init(), "Storage.Init() should not return nil")

	newLogs, err := strg.Logs()
	must(t, err, "Storage.Logs():")
	newLogIDSet := logIDSetFromLogs(newLogs)

	if !logIDSet.Equal(newLogIDSet) {
		t.Fatalf("Missmatch logs: expect %+v, but %+v", logIDSet, newLogIDSet)
	}
}
