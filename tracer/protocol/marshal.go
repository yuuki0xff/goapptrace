package protocol

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func marshalUint64(w io.Writer, val uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], val)
	w.Write(data[:])
}
func unmarshalUint64(r io.Reader) (uint64, error) {
	var data [8]byte
	n, err := r.Read(data[:])
	if err != nil {
		return 0, err
	}
	if n != len(data) {
		return 0, errors.New("lack of length")
	}
	return binary.BigEndian.Uint64(data[:]), nil
}

func marshalByteSlice(w io.Writer, data []byte) {
	marshalUint64(w, uint64(len(data)))
	w.Write(data)
}
func unmarshalByteSlice(r io.Reader) ([]byte, error) {
	length, err := unmarshalUint64(r)
	if err != nil {
		return nil, err
	}

	data := make([]byte, length)
	n, err := r.Read(data)
	if err != nil {
		return nil, err
	}
	if uint64(n) != length {
		return nil, errors.New("lack of length")
	}
	return data, nil
}

func marshalString(w io.Writer, str string) {
	marshalUint64(w, uint64(len(str)))
	w.Write([]byte(str))
}
func unmarshalString(r io.Reader) (string, error) {
	length, err := unmarshalUint64(r)
	if err != nil {
		return "", err
	}

	binstr := make([]byte, length)
	n, err := r.Read(binstr)
	if err != nil {
		return "", err
	}
	if uint64(n) != length {
		return "", errors.New("lack of length")
	}
	return string(binstr), nil
}

func marshalFuncID(w io.Writer, fid logutil.FuncID) {
	marshalUint64(w, uint64(fid))
}
func unmarshalFuncID(r io.Reader) (logutil.FuncID, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.FuncID(0), err
	}
	return logutil.FuncID(val), nil
}

func marshalRawFuncLogID(w io.Writer, id logutil.RawFuncLogID) {
	marshalUint64(w, uint64(id))
}
func unmarshalRawFuncLogID(r io.Reader) (logutil.RawFuncLogID, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.RawFuncLogID(0), err
	}
	return logutil.RawFuncLogID(val), nil
}

func marshalFuncStatusID(w io.Writer, fsid logutil.FuncStatusID) {
	marshalUint64(w, uint64(fsid))
}
func unmarshalFuncStatusID(r io.Reader) (logutil.FuncStatusID, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.FuncStatusID(0), err
	}
	return logutil.FuncStatusID(val), nil
}

func marshalFuncSymbolSlice(w io.Writer, funcs []*logutil.FuncSymbol) {
	marshalUint64(w, uint64(len(funcs)))
	for i := range funcs {
		marshalFuncSymbol(w, funcs[i])
	}
}
func unmarshalFuncSymbolSlice(r io.Reader) ([]*logutil.FuncSymbol, error) {
	length, err := unmarshalUint64(r)
	if err != nil {
		return nil, err
	}

	funcs := make([]*logutil.FuncSymbol, length)
	for i := range funcs {
		funcs[i], err = unmarshalFuncSymbol(r)
		if err != nil {
			return nil, err
		}
	}
	return funcs, nil
}

func marshalFuncSymbol(w io.Writer, s *logutil.FuncSymbol) {
	marshalFuncID(w, s.ID)
	marshalString(w, s.Name)
	marshalString(w, s.File)
	marshalUint64(w, uint64(s.Entry))
}
func unmarshalFuncSymbol(r io.Reader) (*logutil.FuncSymbol, error) {
	s := &logutil.FuncSymbol{}
	s.ID, _ = unmarshalFuncID(r)
	s.Name, _ = unmarshalString(r)
	s.File, _ = unmarshalString(r)
	ptr, _ := unmarshalUint64(r)
	s.Entry = uintptr(ptr)
	return s, nil
}

func marshalFuncStatusSlice(w io.Writer, status []*logutil.FuncStatus) {
	marshalUint64(w, uint64(len(status)))
	for i := range status {
		marshalFuncStatus(w, status[i])
	}
}
func unmarshalFuncStatusSlice(r io.Reader) ([]*logutil.FuncStatus, error) {
	length, err := unmarshalUint64(r)
	if err != nil {
		return nil, err
	}

	funcs := make([]*logutil.FuncStatus, length)
	for i := range funcs {
		funcs[i], err = unmarshalFuncStatus(r)
		if err != nil {
			return nil, err
		}
	}
	return funcs, nil
}

func marshalFuncStatus(w io.Writer, s *logutil.FuncStatus) {
	marshalFuncStatusID(w, s.ID)
	marshalFuncID(w, s.Func)
	marshalUint64(w, s.Line)
	marshalUint64(w, uint64(s.PC))
}
func unmarshalFuncStatus(r io.Reader) (*logutil.FuncStatus, error) {
	s := &logutil.FuncStatus{}
	s.ID, _ = unmarshalFuncStatusID(r)
	s.Func, _ = unmarshalFuncID(r)
	s.Line, _ = unmarshalUint64(r)
	ptr, _ := unmarshalUint64(r)
	s.PC = uintptr(ptr)
	return s, nil
}

func marshalFuncStatusIDSlice(w io.Writer, slice []logutil.FuncStatusID) {
	marshalUint64(w, uint64(len(slice)))
	for i := range slice {
		marshalFuncStatusID(w, slice[i])
	}
}
func unmarshalFuncStatusIDSlice(r io.Reader) ([]logutil.FuncStatusID, error) {
	length, err := unmarshalUint64(r)
	if err != nil {
		return nil, err
	}

	slice := make([]logutil.FuncStatusID, length)
	for i := range slice {
		slice[i], err = unmarshalFuncStatusID(r)
		if err != nil {
			return nil, err
		}
	}
	return slice, nil
}

func marshalGID(w io.Writer, gid logutil.GID) {
	marshalUint64(w, uint64(gid))
}
func unmarshalGID(r io.Reader) (logutil.GID, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.GID(0), err
	}
	return logutil.GID(val), nil
}

func marshalTxID(w io.Writer, id logutil.TxID) {
	marshalUint64(w, uint64(id))
}
func unmarshalTxID(r io.Reader) (logutil.TxID, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.TxID(0), err
	}
	return logutil.TxID(val), nil
}

func marshalTime(w io.Writer, time logutil.Time) {
	marshalUint64(w, uint64(time))
}
func unmarshalTime(r io.Reader) (logutil.Time, error) {
	val, err := unmarshalUint64(r)
	if err != nil {
		return logutil.Time(0), err
	}
	return logutil.Time(val), nil
}

func marshalTagName(w io.Writer, tag logutil.TagName) {
	marshalString(w, string(tag))
}
func unmarshalTagName(r io.Reader) (logutil.TagName, error) {
	str, err := unmarshalString(r)
	if err != nil {
		return logutil.TagName(""), err
	}
	return logutil.TagName(str), nil
}
func marshalRawFuncLog(w io.Writer, r *logutil.RawFuncLog) {
	marshalRawFuncLogID(w, r.ID)
	marshalTagName(w, r.Tag)
	marshalTime(w, r.Timestamp)
	marshalFuncStatusIDSlice(w, r.Frames)
	marshalGID(w, r.GID)
	marshalTxID(w, r.TxID)
}
func unmarshalRawFuncLog(r io.Reader) (*logutil.RawFuncLog, error) {
	fl := &logutil.RawFuncLog{}
	fl.ID, _ = unmarshalRawFuncLogID(r)
	fl.Tag, _ = unmarshalTagName(r)
	fl.Timestamp, _ = unmarshalTime(r)
	fl.Frames, _ = unmarshalFuncStatusIDSlice(r)
	fl.GID, _ = unmarshalGID(r)
	fl.TxID, _ = unmarshalTxID(r)
	return fl, nil
}
