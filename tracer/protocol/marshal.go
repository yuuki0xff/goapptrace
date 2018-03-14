package protocol

import (
	"encoding/binary"

	"github.com/yuuki0xff/goapptrace/tracer/types"
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
	binary.BigEndian.PutUint32(buf[:4], val)
	return 4
}
func unmarshalUint32(buf []byte) (uint32, int64) {
	return binary.BigEndian.Uint32(buf[:4]), 4
}
func marshalUint8(buf []byte, val uint8) int64 {
	buf[0] = val
	return 1
}
func unmarshalUint8(buf []byte) (uint8, int64) {
	return buf[0], 1
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
func sizeString(str string) int64 {
	return int64(8 + len(str))
}

func marshalRawFuncLogID(buf []byte, id types.RawFuncLogID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalRawFuncLogID(buf []byte) (types.RawFuncLogID, int64) {
	val, n := unmarshalUint64(buf)
	return types.RawFuncLogID(val), n
}

func marshalGoLineID(buf []byte, fsid types.GoLineID) int64 {
	return marshalUint64(buf, uint64(fsid))
}
func unmarshalGoLineID(buf []byte) (types.GoLineID, int64) {
	val, n := unmarshalUint64(buf)
	return types.GoLineID(val), n
}

func marshalStringSlice(buf []byte, strs []string) int64 {
	var total int64

	n := marshalUint64(buf[total:], uint64(len(strs)))
	total += n
	for i := range strs {
		n = marshalString(buf[total:], strs[i])
		total += n
	}
	return total
}
func unmarshalStringSlice(buf []byte) ([]string, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	strs := make([]string, length)
	for i := range strs {
		strs[i], n = unmarshalString(buf[total:])
		total += n
	}
	return strs, total
}
func sizeStringSlice(strs []string) int64 {
	total := int64(8) // slice length
	for i := range strs {
		total += sizeString(strs[i])
	}
	return total
}

func marshalGoModuleSlice(buf []byte, mods []types.GoModule) int64 {
	var total int64

	n := marshalUint64(buf[total:], uint64(len(mods)))
	total += n
	for i := range mods {
		n = marshalGoModule(buf[total:], mods[i])
		total += n
	}
	return total
}
func unmarshalGoModuleSlice(buf []byte) ([]types.GoModule, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	mods := make([]types.GoModule, length)
	for i := range mods {
		mods[i], n = unmarshalGoModule(buf[total:])
		total += n
	}
	return mods, total
}
func sizeGoModuleSlice(mods []types.GoModule) int64 {
	total := int64(8) // 8 is bytes of slice length (int64)
	for i := range mods {
		total += sizeGoModule(mods[i])
	}
	return total
}

func marshalGoModule(buf []byte, mod types.GoModule) int64 {
	var total int64
	total += marshalString(buf[total:], mod.Name)
	total += marshalUintptr(buf[total:], mod.MinPC)
	total += marshalUintptr(buf[total:], mod.MaxPC)
	return total
}
func unmarshalGoModule(buf []byte) (types.GoModule, int64) {
	var total int64
	var n int64
	var mod types.GoModule
	mod.Name, n = unmarshalString(buf[total:])
	total += n
	mod.MinPC, n = unmarshalUintptr(buf[total:])
	total += n
	mod.MaxPC, n = unmarshalUintptr(buf[total:])
	total += n
	return mod, total
}
func sizeGoModule(mod types.GoModule) int64 {
	total := sizeString(mod.Name) // Name
	total += 8                    // MinPC
	total += 8                    // MaxPC
	return total
}

func marshalGoFuncSlice(buf []byte, funcs []types.GoFunc) int64 {
	var total int64

	n := marshalUint64(buf[total:], uint64(len(funcs)))
	total += n
	for i := range funcs {
		n = marshalGoFunc(buf[total:], funcs[i])
		total += n
	}
	return total
}
func unmarshalGoFuncSlice(buf []byte) ([]types.GoFunc, int64) {
	var total int64

	length, n := unmarshalUint64(buf)
	total += n

	funcs := make([]types.GoFunc, length)
	for i := range funcs {
		funcs[i], n = unmarshalGoFunc(buf[total:])
		total += n
	}
	return funcs, total
}
func sizeGoFuncSlice(funcs []types.GoFunc) int64 {
	total := int64(8) // slice length
	for i := range funcs {
		total += sizeGoFunc(funcs[i])
	}
	return total
}

func marshalGoFunc(buf []byte, s types.GoFunc) int64 {
	var total int64
	total += marshalString(buf[total:], s.Name)
	total += marshalUint64(buf[total:], uint64(s.Entry))
	return total
}
func unmarshalGoFunc(buf []byte) (types.GoFunc, int64) {
	var total int64
	var n int64

	s := types.GoFunc{}
	total += n
	s.Name, n = unmarshalString(buf[total:])
	total += n
	ptr, n := unmarshalUint64(buf[total:])
	total += n
	s.Entry = uintptr(ptr)
	return s, total
}
func sizeGoFunc(fn types.GoFunc) int64 {
	total := int64(8)            // Entry
	total += sizeString(fn.Name) // Name
	return total
}

func marshalGoLineSlice(buf []byte, line []types.GoLine) int64 {
	total := marshalUint64(buf, uint64(len(line)))
	for i := range line {
		total += marshalGoLine(buf[total:], line[i])
	}
	return total
}
func unmarshalGoLineSlice(buf []byte) ([]types.GoLine, int64) {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	line := make([]types.GoLine, length)
	for i := range line {
		line[i], n = unmarshalGoLine(buf[total:])
		total += n
	}
	return line, total
}
func sizeGoLineSlice(lines []types.GoLine) int64 {
	total := int64(8) // slice length
	for i := range lines {
		total += sizeGoLine(lines[i])
	}
	return total
}

//go:nosplit
func marshalFileID(buf []byte, id types.FileID) int64 {
	return marshalUint32(buf, uint32(id))
}

//go:nosplit
func unmarshalFileID(buf []byte) (types.FileID, int64) {
	id, n := unmarshalUint32(buf)
	return types.FileID(id), n
}
func sizeFileID() int64 {
	return 4
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

func marshalGoLine(buf []byte, s types.GoLine) int64 {
	var total int64
	total += marshalUintptr(buf[total:], s.PC)
	total += marshalFileID(buf[total:], s.FileID)
	total += marshalUint32(buf[total:], s.Line)
	return total
}
func unmarshalGoLine(buf []byte) (types.GoLine, int64) {
	var total int64
	var n int64

	s := types.GoLine{}
	s.PC, n = unmarshalUintptr(buf[total:])
	total += n
	s.FileID, n = unmarshalFileID(buf[total:])
	total += n
	s.Line, n = unmarshalUint32(buf[total:])
	total += n
	return s, total
}
func sizeGoLine(line types.GoLine) int64 {
	total := int64(8)     // PC
	total += sizeFileID() // FileID
	total += 4            // Line
	return total
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

func marshalGID(buf []byte, gid types.GID) int64 {
	return marshalUint64(buf, uint64(gid))
}
func unmarshalGID(buf []byte) (types.GID, int64) {
	val, n := unmarshalUint64(buf)
	return types.GID(val), n
}

func marshalTxID(buf []byte, id types.TxID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalTxID(buf []byte) (types.TxID, int64) {
	val, n := unmarshalUint64(buf)
	return types.TxID(val), n
}

func marshalTime(buf []byte, time types.Time) int64 {
	return marshalUint64(buf, uint64(time))
}
func unmarshalTime(buf []byte) (types.Time, int64) {
	val, n := unmarshalUint64(buf)
	return types.Time(val), n
}

func marshalTagName(buf []byte, tag types.TagName) int64 {
	buf[0] = byte(tag)
	return 1
}
func unmarshalTagName(buf []byte) (types.TagName, int64) {
	return types.TagName(buf[0]), 1
}
func marshalRawFuncLog(buf []byte, r *types.RawFuncLog) int64 {
	total := marshalRawFuncLogID(buf, r.ID)
	total += marshalTagName(buf[total:], r.Tag)
	total += marshalTime(buf[total:], r.Timestamp)
	total += marshalUintptrSlice(buf[total:], r.Frames)
	total += marshalGID(buf[total:], r.GID)
	total += marshalTxID(buf[total:], r.TxID)
	return total
}
func unmarshalRawFuncLog(buf []byte) (*types.RawFuncLog, int64) {
	var total int64
	var n int64

	fl := &types.RawFuncLog{}
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
