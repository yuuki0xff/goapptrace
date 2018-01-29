package protocol

import (
	"encoding/binary"
	"io"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	. "github.com/yuuki0xff/goapptrace/tracer/util"
)

var (
	trueBytes  = []byte{1}
	falseBytes = []byte{0}
)

func marshalBool(w io.Writer, val bool) {
	if val {
		MustWrite(w, trueBytes)
	} else {
		MustWrite(w, falseBytes)
	}
}
func unmarshalBool(r io.Reader) bool {
	var data [1]byte
	MustRead(r, data[:])
	return data[0] != 0
}

func marshalUint64(w io.Writer, val uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], val)
	MustWrite(w, data[:])
}
func unmarshalUint64(r io.Reader) uint64 {
	var data [8]byte
	MustRead(r, data[:])
	return binary.BigEndian.Uint64(data[:])
}

func marshalString(w io.Writer, str string) {
	marshalUint64(w, uint64(len(str)))
	MustWrite(w, []byte(str))
}
func unmarshalString(r io.Reader) string {
	length := unmarshalUint64(r)
	binstr := make([]byte, length)
	MustRead(r, binstr)
	return string(binstr)
}

func marshalFuncID(w io.Writer, fid logutil.FuncID) {
	marshalUint64(w, uint64(fid))
}
func unmarshalFuncID(r io.Reader) logutil.FuncID {
	val := unmarshalUint64(r)
	return logutil.FuncID(val)
}

func marshalRawFuncLogID(w io.Writer, id logutil.RawFuncLogID) {
	marshalUint64(w, uint64(id))
}
func unmarshalRawFuncLogID(r io.Reader) logutil.RawFuncLogID {
	val := unmarshalUint64(r)
	return logutil.RawFuncLogID(val)
}

func marshalFuncStatusID(w io.Writer, fsid logutil.FuncStatusID) {
	marshalUint64(w, uint64(fsid))
}
func unmarshalFuncStatusID(r io.Reader) logutil.FuncStatusID {
	val := unmarshalUint64(r)
	return logutil.FuncStatusID(val)
}

func marshalFuncSymbolSlice(w io.Writer, funcs []*logutil.FuncSymbol) {
	marshalUint64(w, uint64(len(funcs)))
	for i := range funcs {
		marshalBool(w, funcs[i] != nil)
		if funcs[i] != nil {
			marshalFuncSymbol(w, funcs[i])
		}
	}
}
func unmarshalFuncSymbolSlice(r io.Reader) []*logutil.FuncSymbol {
	length := unmarshalUint64(r)
	funcs := make([]*logutil.FuncSymbol, length)
	for i := range funcs {
		isNonNil := unmarshalBool(r)
		if isNonNil {
			funcs[i] = unmarshalFuncSymbol(r)
		}
	}
	return funcs
}

func marshalFuncSymbol(w io.Writer, s *logutil.FuncSymbol) {
	marshalFuncID(w, s.ID)
	marshalString(w, s.Name)
	marshalString(w, s.File)
	marshalUint64(w, uint64(s.Entry))
}
func unmarshalFuncSymbol(r io.Reader) *logutil.FuncSymbol {
	s := &logutil.FuncSymbol{}
	s.ID = unmarshalFuncID(r)
	s.Name = unmarshalString(r)
	s.File = unmarshalString(r)
	ptr := unmarshalUint64(r)
	s.Entry = uintptr(ptr)
	return s
}

func marshalFuncStatusSlice(w io.Writer, status []*logutil.FuncStatus) {
	marshalUint64(w, uint64(len(status)))
	for i := range status {
		marshalBool(w, status[i] != nil)
		if status[i] != nil {
			marshalFuncStatus(w, status[i])
		}
	}
}
func unmarshalFuncStatusSlice(r io.Reader) []*logutil.FuncStatus {
	length := unmarshalUint64(r)
	funcs := make([]*logutil.FuncStatus, length)
	for i := range funcs {
		isNonNil := unmarshalBool(r)
		if isNonNil {
			funcs[i] = unmarshalFuncStatus(r)
		}
	}
	return funcs
}

func marshalFuncStatus(w io.Writer, s *logutil.FuncStatus) {
	marshalFuncStatusID(w, s.ID)
	marshalFuncID(w, s.Func)
	marshalUint64(w, s.Line)
	marshalUint64(w, uint64(s.PC))
}
func unmarshalFuncStatus(r io.Reader) *logutil.FuncStatus {
	s := &logutil.FuncStatus{}
	s.ID = unmarshalFuncStatusID(r)
	s.Func = unmarshalFuncID(r)
	s.Line = unmarshalUint64(r)
	ptr := unmarshalUint64(r)
	s.PC = uintptr(ptr)
	return s
}

func marshalFuncStatusIDSlice(w io.Writer, slice []logutil.FuncStatusID) {
	marshalUint64(w, uint64(len(slice)))
	for i := range slice {
		marshalFuncStatusID(w, slice[i])
	}
}
func unmarshalFuncStatusIDSlice(r io.Reader) []logutil.FuncStatusID {
	length := unmarshalUint64(r)
	slice := make([]logutil.FuncStatusID, length)
	for i := range slice {
		slice[i] = unmarshalFuncStatusID(r)
	}
	return slice
}

func marshalGID(w io.Writer, gid logutil.GID) {
	marshalUint64(w, uint64(gid))
}
func unmarshalGID(r io.Reader) logutil.GID {
	val := unmarshalUint64(r)
	return logutil.GID(val)
}

func marshalTxID(w io.Writer, id logutil.TxID) {
	marshalUint64(w, uint64(id))
}
func unmarshalTxID(r io.Reader) logutil.TxID {
	val := unmarshalUint64(r)
	return logutil.TxID(val)
}

func marshalTime(w io.Writer, time logutil.Time) {
	marshalUint64(w, uint64(time))
}
func unmarshalTime(r io.Reader) logutil.Time {
	val := unmarshalUint64(r)
	return logutil.Time(val)
}

func marshalTagName(w io.Writer, tag logutil.TagName) {
	marshalString(w, string(tag))
}
func unmarshalTagName(r io.Reader) logutil.TagName {
	str := unmarshalString(r)
	return logutil.TagName(str)
}
func marshalRawFuncLog(w io.Writer, r *logutil.RawFuncLog) {
	marshalRawFuncLogID(w, r.ID)
	marshalTagName(w, r.Tag)
	marshalTime(w, r.Timestamp)
	marshalFuncStatusIDSlice(w, r.Frames)
	marshalGID(w, r.GID)
	marshalTxID(w, r.TxID)
}
func unmarshalRawFuncLog(r io.Reader) *logutil.RawFuncLog {
	fl := &logutil.RawFuncLog{}
	fl.ID = unmarshalRawFuncLogID(r)
	fl.Tag = unmarshalTagName(r)
	fl.Timestamp = unmarshalTime(r)
	fl.Frames = unmarshalFuncStatusIDSlice(r)
	fl.GID = unmarshalGID(r)
	fl.TxID = unmarshalTxID(r)
	return fl
}
