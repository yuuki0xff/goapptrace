package storage

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const (
	// goapptraceによって作成されたディレクトリとファイルの、デフォルトのパーミッション
	DefaultDirPerm  = 0777
	DefaultFilePerm = 0666

	DefaultCompressionLevel = gzip.BestSpeed
)

// ディレクトリ構造を抽象化する。
type DirLayout struct {
	Root string
}

// ディレクトリを初期化する。
// 必要なディレクトリが存在しない場合、作成する。
func (d DirLayout) Init() error {
	// create Root dir
	if err := os.MkdirAll(d.Root, DefaultDirPerm); err != nil {
		return err
	}

	if d.InfoFile().Exists() {
		// check whether that data format have compatible.
		data, err := d.InfoFile().ReadAll()
		if err != nil {
			return err
		}
		var info Info
		if err := json.Unmarshal(data, &info); err != nil {
			return err
		}
		if !info.IsCompatible() {
			return fmt.Errorf("data format is not compatible")
		}
	} else {
		// write the current data format version.
		info := Info{
			MajorVersion: MajorVersion,
			MinorVersion: MinorVersion,
		}
		data, err := json.Marshal(&info)
		if err != nil {
			return err
		}

		w, err := d.InfoFile().OpenWriteOnly()
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}

	// create subdirectories
	if err := os.MkdirAll(d.MetaDir(), DefaultDirPerm); err != nil {
		return err
	}
	if err := os.MkdirAll(d.DataDir(), DefaultDirPerm); err != nil {
		return err
	}
	return nil
}

// infoファイルを返す
func (d DirLayout) InfoFile() File {
	return File(path.Join(d.Root, "info.json.gz"))
}

// メタデータファイルが格納されるディレクトリのパスを返す。
func (d DirLayout) MetaDir() string {
	return path.Join(d.Root, "meta")
}

// ログファイル化格納されるディレクトリのパスを返す。
func (d DirLayout) DataDir() string {
	return path.Join(d.Root, "data")
}

// ファイル名(basename)からLogIDに変換する。
func (d DirLayout) Fname2LogID(fname string) (id LogID, ok bool) {
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

// 指定したLogIDのメタデータファイルを返す。
func (d DirLayout) MetaFile(id LogID) File {
	return File(path.Join(d.MetaDir(), id.Hex()+".meta.json.gz"))
}

// 指定したLogIDのRawFuncLogファイルを返す。
func (d DirLayout) RawFuncLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.rawfunc.log.gz", id.Hex(), n)))
}

// 指定したLogIDのFuncLogファイルを返す。
func (d DirLayout) FuncLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.func.log.gz", id.Hex(), n)))
}

// 指定したLogIDのGoroutineLogファイルを返す。
func (d DirLayout) GoroutineLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.goroutine.log.gz", id.Hex(), n)))
}

// 指定したLogIDのSymbolファイルを返す。
func (d DirLayout) SymbolFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.symbol.gz", id.Hex())))
}

// 指定したLogIDのIndexファイルを返す。
func (d DirLayout) IndexFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.index.gz", id.Hex())))
}

// ファイル操作を抽象化する。
type File string

// このファイルを削除する。
func (f File) Remove() error {
	return os.Remove(string(f))
}

// このファイルが存在するならtrueを返す。
func (f File) Exists() bool {
	_, err := os.Stat(string(f))
	return err == nil
}

// このファイルの実際のサイズを返す。
func (f File) Size() (int64, error) {
	stat, err := os.Stat(string(f))
	if err != nil {
		return 0, nil
	}
	return stat.Size(), err
}

// ReadOnlyモードで開く。
func (f File) OpenReadOnly() (io.ReadCloser, error) {
	file, err := os.Open(string(f))
	if err != nil {
		return nil, fmt.Errorf("cannot open %s for reading: %s", string(f), err)
	}
	return gzip.NewReader(file)
}

// WriteOnlyモードで開く。
// 既存のデータがあった場合、開いた直後にtruncateされる。
func (f File) OpenWriteOnly() (io.WriteCloser, error) {
	file, err := os.OpenFile(string(f), os.O_CREATE|os.O_WRONLY, DefaultFilePerm)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s for writing: %s", string(f), err)
	}
	return gzip.NewWriterLevel(file, DefaultCompressionLevel)
}

// AppendOnlyモードで開く。
func (f File) OpenAppendOnly() (io.WriteCloser, error) {
	file, err := os.OpenFile(string(f), os.O_CREATE|os.O_WRONLY|os.O_APPEND, DefaultFilePerm)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s for appending: %s", string(f), err)
	}
	return gzip.NewWriterLevel(file, DefaultCompressionLevel)
}

// ファイルから全て読み込み、[]byteを返す。
func (f File) ReadAll() ([]byte, error) {
	file, err := f.OpenReadOnly()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(file)
}
