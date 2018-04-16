package storage

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
)

type FileReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}
type FileWriter interface {
	io.Writer
	io.Seeker
	io.Closer
}

// ディレクトリ構造を抽象化する。
type DirLayout struct {
	Root string
}

// ディレクトリを初期化する。
// 必要なディレクトリが存在しない場合、作成する。
func (d DirLayout) Init() error {
	// create Root dir
	if err := d.mkdir(d.Root); err != nil {
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
			return errors.New("data format is not compatible")
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
	if err := d.mkdir(d.MetaDir()); err != nil {
		return err
	}
	if err := d.mkdir(d.DataDir()); err != nil {
		return err
	}
	return nil
}

// infoファイルを返す
func (d DirLayout) InfoFile() File {
	return File(path.Join(d.Root, "info.json"))
}

func (d DirLayout) TracersFile() File {
	return File(path.Join(d.Root, "tracers.json"))
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
	if !strings.HasSuffix(fname, ".meta.json") {
		return
	}
	strid := strings.TrimSuffix(fname, ".meta.json")
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
	return File(path.Join(d.MetaDir(), id.Hex()+".meta.json"))
}

// 指定したLogIDのRawFuncLogファイルを返す。
func (d DirLayout) RawFuncLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.rawfunc.log", id.Hex(), n)))
}

// 指定したLogIDのFuncLogファイルを返す。
func (d DirLayout) FuncLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.func.log", id.Hex(), n)))
}

// 指定したLogIDのGoroutineLogファイルを返す。
func (d DirLayout) GoroutineLogFile(id LogID, n int64) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.%d.goroutine.log", id.Hex(), n)))
}

// 指定したLogIDのSymbolファイルを返す。
func (d DirLayout) SymbolFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.symbol", id.Hex())))
}

// 指定したLogIDのIndexファイルを返す。
func (d DirLayout) IndexFile(id LogID) File {
	return File(path.Join(d.DataDir(), fmt.Sprintf("%s.index", id.Hex())))
}

func (d DirLayout) mkdir(dir string) error {
	return os.MkdirAll(dir, config.DefaultDirPerm)
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
func (f File) OpenReadOnly() (FileReader, error) {
	file, err := os.Open(string(f))
	return file, errors.Wrapf(err, "cannot open %s for reading: %s", string(f))
}

// WriteOnlyモードで開く。
// 既存のデータがあった場合、開いた直後にtruncateされる。
func (f File) OpenWriteOnly() (FileWriter, error) {
	file, err := f.openFile(string(f), os.O_CREATE|os.O_WRONLY)
	return file, errors.Wrapf(err, "cannot open %s for writing: %s", string(f))
}

// AppendOnlyモードで開く。
func (f File) OpenAppendOnly() (FileWriter, error) {
	file, err := f.openFile(string(f), os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	return file, errors.Wrapf(err, "cannot open %s for appending: %s", string(f))
}

func (f File) RenameTo(to File) error {
	return os.Rename(string(f), string(to))
}

// ファイルから全て読み込み、[]byteを返す。
func (f File) ReadAll() ([]byte, error) {
	file, err := f.OpenReadOnly()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(file)
}

// ファイルにdataを書き込む。書き込みはatomicに行われる。
func (f File) WriteAll(data []byte) error {
	newf := f.new()
	err := ioutil.WriteFile(string(newf), data, 0600)
	if err != nil {
		return err
	}
	return newf.RenameTo(f)
}

func (f File) openFile(name string, flag int) (*os.File, error) {
	return os.OpenFile(name, flag, config.DefaultFilePerm)
}
func (f File) new() File {
	return File(string(f) + ".new." + strconv.FormatUint(rand.Uint64(), 10))
}
