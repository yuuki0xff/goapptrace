package storage

import (
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type DirLayout struct {
	Root string
}

func (d DirLayout) Init() error {
	return os.MkdirAll(d.Root, 0666)
}

func (d DirLayout) MetaDir() string {
	return path.Join(d.Root, "meta")
}
func (d DirLayout) DataDir() string {
	return path.Join(d.Root, "data")
}
func (d DirLayout) MetaID(fname string) (id LogID, ok bool) {
	if !strings.HasSuffix(fname, ".meta.json.gz") {
		return
	}
	strid := strings.TrimSuffix(fname, ".meta.json.gz")
	binid, err := hex.DecodeString(strid)
	if err != nil {
		return
	}
	if len(binid) != len(id) {
		return
	}

	copy(id[:], binid)
	ok = true
	return
}

func (d DirLayout) MetaFile(id LogID) File {
	return File(path.Join(d.MetaDir(), id.Hex()+".meta.json.gz"))
}
func (d DirLayout) FuncLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.func.log.gz", id.Hex(), n)))
}
func (d DirLayout) SymbolFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.symbol.gz", id.Hex())))
}
func (d DirLayout) IndexFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.index.gz", id.Hex())))
}

type File string

func (f File) Exists() bool {
	_, err := os.Stat(string(f))
	return os.IsExist(err)
}
func (f File) Size() (int64, error) {
	stat, err := os.Stat(string(f))
	if err != nil {
		return 0, nil
	}
	return stat.Size(), err
}
func (f File) OpenReadOnly() (io.ReadCloser, error) {
	file, err := os.Open(string(f))
	if err != nil {
		return nil, err
	}
	return gzip.NewReader(file)
}
func (f File) OpenWriteOnly() (io.WriteCloser, error) {
	file, err := os.OpenFile(string(f), os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return gzip.NewWriterLevel(file, gzip.BestCompression)
}
func (f File) OpenAppendOnly() (io.WriteCloser, error) {
	file, err := os.OpenFile(string(f), os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return gzip.NewWriterLevel(file, gzip.BestCompression)
}
func (f File) ReadAll() ([]byte, error) {
	file, err := f.OpenReadOnly()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(file)
}
