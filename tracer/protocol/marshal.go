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

func marshalUint32(buf []byte, val uint32) int64 {
	binary.BigEndian.PutUint32(buf[:8], val)
	return 8
}
func unmarshalUint32(buf []byte) (uint32, int64) {
	return binary.BigEndian.Uint32(buf[:8]), 8
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

func marshalRawFuncLogID(buf []byte, id logutil.RawFuncLogID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalRawFuncLogID(buf []byte) (logutil.RawFuncLogID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.RawFuncLogID(val), n
}

func marshalGoLineID(buf []byte, fsid logutil.GoLineID) int64 {
	return marshalUint64(buf, uint64(fsid))
}
func unmarshalGoLineID(buf []byte) (logutil.GoLineID, int64) {
	val, n := unmarshalUint64(buf)
	return logutil.GoLineID(val), n
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
	total += marshalString(buf[total:], s.Name)
	total += marshalUint64(buf[total:], uint64(s.Entry))
	return total
}
func unmarshalGoFunc(buf []byte) (*logutil.GoFunc, int64) {
	var total int64
	var n int64

	s := &logutil.GoFunc{}
	total += n
	s.Name, n = unmarshalString(buf[total:])
	total += n
	ptr, n := unmarshalUint64(buf[total:])
	total += n
	s.Entry = uintptr(ptr)
	return s, total
}

func marshalGoLineSlice(buf []byte, status []*logutil.GoLine) int64 {
	total := marshalUint64(buf, uint64(len(status)))
	for i := range status {
		total += marshalBool(buf[total:], status[i] != nil)
		if status[i] != nil {
			total += marshalGoLine(buf[total:], status[i])
		}
	}
	return total
}
func unmarshalGoLineSlice(buf []byte) ([]*logutil.GoLine, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	funcs := make([]*logutil.GoLine, length)
	for i := range funcs {
		isNonNil, n := unmarshalBool(buf[total:])
		total += n
		if isNonNil {
			funcs[i], n = unmarshalGoLine(buf[total:])
			total += n
		}
	}
	return funcs, total
}

//go:nosplit
func marshalFileID(buf []byte, id logutil.FileID) int64 {
	return marshalUint64(buf, uint64(id))
}

//go:nosplit
func unmarshalFileID(buf []byte) (logutil.FileID, int64) {
	id, n := unmarshalUint64(buf)
	return logutil.FileID(id), n
}

//go:nosplit
func marshalUintptr(buf []byte, ptr uintptr) int64 {
	return marshalUint64(buf, uint64(ptr))
}

//go:nosplit
func unmarshalUintptr(buf []byte) (uintptr, int64) {
	ptr, n := unmarshalUint64(buf)
	return uintptr(ptr), n
}

func marshalGoLine(buf []byte, s *logutil.GoLine) int64 {
	var total int64
	total += marshalUintptr(buf[total:], s.PC)
	total += marshalFileID(buf[total:], s.FileID)
	total += marshalUint32(buf[total:], s.Line)
	return total
}
func unmarshalGoLine(buf []byte) (*logutil.GoLine, int64) {
	var total int64
	var n int64

	s := &logutil.GoLine{}
	s.PC, n = unmarshalUintptr(buf[total:])
	total += n
	s.FileID, n = unmarshalFileID(buf[total:])
	total += n
	s.Line, n = unmarshalUint32(buf[total:])
	total += n
	return s, total
}

func marshalUintptrSlice(buf []byte, slice []uintptr) int64 {
	total := marshalUint64(buf, uint64(len(slice)))
	for i := range slice {
		total += marshalUint64(buf[total:], uint64(slice[i]))
	}
	return total
}
func unmarshalUintptrSlice(buf []byte) ([]uintptr, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	slice := make([]uintptr, length)
	for i := range slice {
		ptr, n := unmarshalUint64(buf[total:])
		slice[i] = uintptr(ptr)
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
	buf[0] = byte(tag)
	return 1
}
func unmarshalTagName(buf []byte) (logutil.TagName, int64) {
	return logutil.TagName(buf[0]), 1
}
func marshalRawFuncLog(buf []byte, r *logutil.RawFuncLog) int64 {
	total := marshalRawFuncLogID(buf, r.ID)
	total += marshalTagName(buf[total:], r.Tag)
	total += marshalTime(buf[total:], r.Timestamp)
	total += marshalUintptrSlice(buf[total:], r.Frames)
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
	fl.Frames, n = unmarshalUintptrSlice(buf[total:])
	total += n
	fl.GID, n = unmarshalGID(buf[total:])
	total += n
	fl.TxID, n = unmarshalTxID(buf[total:])
	total += n
	return fl, total
}
