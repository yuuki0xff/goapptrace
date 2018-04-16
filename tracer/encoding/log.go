package encoding

import (
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func MarshalGoroutine(buf []byte, g *types.Goroutine) int64 {
	total := marshalGID(buf, g.GID)
	total += marshalTime(buf[total:], g.StartTime)
	total += marshalTime(buf[total:], g.EndTime)
	return total
}
func UnmarshalGoroutine(buf []byte, g *types.Goroutine) int64 {
	var total int64
	var n int64

	g.GID, n = unmarshalGID(buf)
	total += n
	g.StartTime, n = unmarshalTime(buf[total:])
	total += n
	g.EndTime, n = unmarshalTime(buf[total:])
	total += n
	return total
}
func SizeGoroutine() int64 {
	var total int64
	total += 8 * 3 // 8byteのフィールドが3個 (GID, StartTime, EndTime)
	return total
}

func MarshalFuncLog(buf []byte, f *types.FuncLog) int64 {
	total := marshalFuncLogID(buf, f.ID)
	total += marshalTime(buf[total:], f.StartTime)
	total += marshalTime(buf[total:], f.EndTime)
	total += marshalFuncLogID(buf[total:], f.ParentID)
	total += marshalUintptrSlice(buf[total:], f.Frames)
	total += marshalGID(buf[total:], f.GID)
	return total
}

// fl.Frames には十分なサイズのバッファが容易されて無ければならない。
func UnmarshalFuncLog(buf []byte, f *types.FuncLog) int64 {
	var total int64
	var n int64

	f.ID, n = unmarshalFuncLogID(buf)
	total += n
	f.StartTime, n = unmarshalTime(buf[total:])
	total += n
	f.EndTime, n = unmarshalTime(buf[total:])
	total += n
	f.ParentID, n = unmarshalFuncLogID(buf[total:])
	total += n
	total += unmarshalUintptrSlice(buf[total:], &f.Frames)
	f.GID, n = unmarshalGID(buf[total:])
	total += n
	return total
}
func SizeFuncLog() int64 {
	var total int64
	total += 8 * 5                        // 8byteのフィールドが5個 (ID, StartTime, EndTime, ParentID, GID)
	total += 8 * (1 + types.MaxStackSize) // スライスが1個 (Frames)
	return total
}

func MarshalRawFuncLog(buf []byte, r *types.RawFuncLog) int64 {
	total := marshalRawFuncLogID(buf, r.ID)
	total += marshalTagName(buf[total:], r.Tag)
	total += marshalTime(buf[total:], r.Timestamp)
	total += marshalUintptrSlice(buf[total:], r.Frames)
	total += marshalGID(buf[total:], r.GID)
	total += marshalTxID(buf[total:], r.TxID)
	return total
}

// fl.Frames には十分なサイズのバッファが容易されて無ければならない。
func UnmarshalRawFuncLog(buf []byte, r *types.RawFuncLog) int64 {
	var total int64
	var n int64

	r.ID, n = unmarshalRawFuncLogID(buf)
	total += n
	r.Tag, n = unmarshalTagName(buf[total:])
	total += n
	r.Timestamp, n = unmarshalTime(buf[total:])
	total += n
	total += unmarshalUintptrSlice(buf[total:], &r.Frames)
	r.GID, n = unmarshalGID(buf[total:])
	total += n
	r.TxID, n = unmarshalTxID(buf[total:])
	total += n
	return total
}
func SizeRawFuncLog() int64 {
	var total int64
	total += 8 * 4                        // 8byteのフィールドが4個 (ID, Timestamp, GID, TxID)
	total += 1                            // 1byteのフィールドが1個 (Tag)
	total += 8 * (1 + types.MaxStackSize) // スライスが1個 (Frames)
	return total
}

func marshalGoModule(buf []byte, mod types.GoModule) int64 {
	var total int64
	total += MarshalString(buf[total:], mod.Name)
	total += MarshalUintptr(buf[total:], mod.MinPC)
	total += MarshalUintptr(buf[total:], mod.MaxPC)
	return total
}
func unmarshalGoModule(buf []byte) (types.GoModule, int64) {
	var total int64
	var n int64
	var mod types.GoModule
	mod.Name, n = UnmarshalString(buf[total:])
	total += n
	mod.MinPC, n = UnmarshalUintptr(buf[total:])
	total += n
	mod.MaxPC, n = UnmarshalUintptr(buf[total:])
	total += n
	return mod, total
}
func sizeGoModule(mod types.GoModule) int64 {
	total := sizeString(mod.Name) // Name
	total += 8                    // MinPC
	total += 8                    // MaxPC
	return total
}

func marshalGoModuleSlice(buf []byte, mods []types.GoModule) int64 {
	var total int64

	n := MarshalUint64(buf[total:], uint64(len(mods)))
	total += n
	for i := range mods {
		n = marshalGoModule(buf[total:], mods[i])
		total += n
	}
	return total
}
func unmarshalGoModuleSlice(buf []byte) ([]types.GoModule, int64) {
	var total int64
	length, n := UnmarshalUint64(buf)
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

func marshalGoFunc(buf []byte, s types.GoFunc) int64 {
	var total int64
	total += MarshalString(buf[total:], s.Name)
	total += MarshalUint64(buf[total:], uint64(s.Entry))
	return total
}
func unmarshalGoFunc(buf []byte) (types.GoFunc, int64) {
	var total int64
	var n int64

	s := types.GoFunc{}
	total += n
	s.Name, n = UnmarshalString(buf[total:])
	total += n
	ptr, n := UnmarshalUint64(buf[total:])
	total += n
	s.Entry = uintptr(ptr)
	return s, total
}
func sizeGoFunc(fn types.GoFunc) int64 {
	total := int64(8)            // Entry
	total += sizeString(fn.Name) // Name
	return total
}

func marshalGoFuncSlice(buf []byte, funcs []types.GoFunc) int64 {
	var total int64

	n := MarshalUint64(buf[total:], uint64(len(funcs)))
	total += n
	for i := range funcs {
		n = marshalGoFunc(buf[total:], funcs[i])
		total += n
	}
	return total
}
func unmarshalGoFuncSlice(buf []byte) ([]types.GoFunc, int64) {
	var total int64

	length, n := UnmarshalUint64(buf)
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

func marshalGoLine(buf []byte, s types.GoLine) int64 {
	var total int64
	total += MarshalUintptr(buf[total:], s.PC)
	total += marshalFileID(buf[total:], s.FileID)
	total += marshalUint32(buf[total:], s.Line)
	return total
}
func unmarshalGoLine(buf []byte) (types.GoLine, int64) {
	var total int64
	var n int64

	s := types.GoLine{}
	s.PC, n = UnmarshalUintptr(buf[total:])
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

func marshalGoLineSlice(buf []byte, line []types.GoLine) int64 {
	total := MarshalUint64(buf, uint64(len(line)))
	for i := range line {
		total += marshalGoLine(buf[total:], line[i])
	}
	return total
}
func unmarshalGoLineSlice(buf []byte) ([]types.GoLine, int64) {
	var total int64
	length, n := UnmarshalUint64(buf)
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
