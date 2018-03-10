package protocol

import (
	"encoding/binary"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func marshalBool(buf []byte, val bool) int64 {
	if val {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	return 1
}
func unmarshalBool(buf []byte) (bool, int64) {
	return buf[0] != 0, 1
}

func marshalUint64(buf []byte, val uint64) int64 {
	binary.BigEndian.PutUint64(buf[:8], val)
	return 8
}
func unmarshalUint64(buf []byte) (uint64, int64) {
	return binary.BigEndian.Uint64(buf[:8]), 8
}

func marshalString(buf []byte, str string) int64 {
	total := marshalUint64(buf, uint64(len(str)))
	total += int64(copy(buf[total:], []byte(str)))
	return total
}
func unmarshalString(buf []byte) (string, int64) {
	length, n := unmarshalUint64(buf)
	buf = buf[n:]
	return string(buf[:length]), n + int64(length)
}

func marshalFuncID(buf []byte, fid logutil.FuncID) int64 {
	return marshalUint64(buf, uint64(fid))
}
func unmarshalFuncID(buf []byte) (logutil.FuncID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.FuncID(val), n
}

func marshalRawFuncLogID(buf []byte, id logutil.RawFuncLogID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalRawFuncLogID(buf []byte) (logutil.RawFuncLogID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.RawFuncLogID(val), n
}

func marshalFuncStatusID(buf []byte, fsid logutil.FuncStatusID) int64 {
	return marshalUint64(buf, uint64(fsid))
}
func unmarshalFuncStatusID(buf []byte) (logutil.FuncStatusID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.FuncStatusID(val), n
}

func marshalGoFuncSlice(buf []byte, funcs []*logutil.GoFunc) int64 {
	var total int64

	n := marshalUint64(buf[total:], uint64(len(funcs)))
	total += n
	for i := range funcs {
		n = marshalBool(buf[total:], funcs[i] != nil)
		total += n
		if funcs[i] != nil {
			n = marshalGoFunc(buf[total:], funcs[i])
			total += n
		}
	}
	return total
}
func unmarshalGoFuncSlice(buf []byte) ([]*logutil.GoFunc, int64) {
	var total int64

	length, n := unmarshalUint64(buf)
	total += n

	funcs := make([]*logutil.GoFunc, length)
	for i := range funcs {
		isNonNil, n := unmarshalBool(buf[total:])
		total += n
		if isNonNil {
			funcs[i], n = unmarshalGoFunc(buf[total:])
			total += n
		}
	}
	return funcs, total
}

func marshalGoFunc(buf []byte, s *logutil.GoFunc) int64 {
	var total int64
	total += marshalFuncID(buf, s.ID)
	total += marshalString(buf[total:], s.Name)
	total += marshalString(buf[total:], s.File)
	total += marshalUint64(buf[total:], uint64(s.Entry))
	return total
}
func unmarshalGoFunc(buf []byte) (*logutil.GoFunc, int64) {
	var total int64
	var n int64

	s := &logutil.GoFunc{}
	s.ID, n = unmarshalFuncID(buf)
	total += n
	s.Name, n = unmarshalString(buf[total:])
	total += n
	s.File, n = unmarshalString(buf[total:])
	total += n
	ptr, n := unmarshalUint64(buf[total:])
	total += n
	s.Entry = uintptr(ptr)
	return s, total
}

func marshalFuncStatusSlice(buf []byte, status []*logutil.FuncStatus) int64 {
	total := marshalUint64(buf, uint64(len(status)))
	for i := range status {
		total += marshalBool(buf[total:], status[i] != nil)
		if status[i] != nil {
			total += marshalFuncStatus(buf[total:], status[i])
		}
	}
	return total
}
func unmarshalFuncStatusSlice(buf []byte) ([]*logutil.FuncStatus, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	funcs := make([]*logutil.FuncStatus, length)
	for i := range funcs {
		isNonNil, n := unmarshalBool(buf[total:])
		total += n
		if isNonNil {
			funcs[i], n = unmarshalFuncStatus(buf[total:])
			total += n
		}
	}
	return funcs, total
}

func marshalFuncStatus(buf []byte, s *logutil.FuncStatus) int64 {
	var total int64
	total += marshalFuncStatusID(buf, s.ID)
	total += marshalFuncID(buf[total:], s.Func)
	total += marshalUint64(buf[total:], s.Line)
	total += marshalUint64(buf[total:], uint64(s.PC))
	return total
}
func unmarshalFuncStatus(buf []byte) (*logutil.FuncStatus, int64) {
	var total int64
	var n int64

	s := &logutil.FuncStatus{}
	s.ID, n = unmarshalFuncStatusID(buf)
	total += n
	s.Func, n = unmarshalFuncID(buf[total:])
	total += n
	s.Line, n = unmarshalUint64(buf[total:])
	total += n
	ptr, n := unmarshalUint64(buf[total:])
	total += n
	s.PC = uintptr(ptr)
	return s, total
}

func marshalFuncStatusIDSlice(buf []byte, slice []logutil.FuncStatusID) int64 {
	total := marshalUint64(buf, uint64(len(slice)))
	for i := range slice {
		total += marshalFuncStatusID(buf[total:], slice[i])
	}
	return total
}
func unmarshalFuncStatusIDSlice(buf []byte) ([]logutil.FuncStatusID, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	slice := make([]logutil.FuncStatusID, length)
	for i := range slice {
		var n int64
		slice[i], n = unmarshalFuncStatusID(buf[total:])
		total += n
	}
	return slice, total
}

func marshalGID(buf []byte, gid logutil.GID) int64 {
	return marshalUint64(buf, uint64(gid))
}
func unmarshalGID(buf []byte) (logutil.GID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.GID(val), n
}

func marshalTxID(buf []byte, id logutil.TxID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalTxID(buf []byte) (logutil.TxID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.TxID(val), n
}

func marshalTime(buf []byte, time logutil.Time) int64 {
	return marshalUint64(buf, uint64(time))
}
func unmarshalTime(buf []byte) (logutil.Time, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.Time(val), n
}

func marshalTagName(buf []byte, tag logutil.TagName) int64 {
	return marshalString(buf, string(tag))
}
func unmarshalTagName(buf []byte) (logutil.TagName, int64) {
	str, n := unmarshalString(buf)
	return logutil.TagName(str), n
}
func marshalRawFuncLog(buf []byte, r *logutil.RawFuncLog) int64 {
	total := marshalRawFuncLogID(buf, r.ID)
	total += marshalTagName(buf[total:], r.Tag)
	total += marshalTime(buf[total:], r.Timestamp)
	total += marshalFuncStatusIDSlice(buf[total:], r.Frames)
	total += marshalGID(buf[total:], r.GID)
	total += marshalTxID(buf[total:], r.TxID)
	return total
}
func unmarshalRawFuncLog(buf []byte) (*logutil.RawFuncLog, int64) {
	var total int64
	var n int64

	fl := &logutil.RawFuncLog{}
	fl.ID, n = unmarshalRawFuncLogID(buf)
	total += n
	fl.Tag, n = unmarshalTagName(buf[total:])
	total += n
	fl.Timestamp, n = unmarshalTime(buf[total:])
	total += n
	fl.Frames, n = unmarshalFuncStatusIDSlice(buf[total:])
	total += n
	fl.GID, n = unmarshalGID(buf[total:])
	total += n
	fl.TxID, n = unmarshalTxID(buf[total:])
	total += n
	return fl, total
}
