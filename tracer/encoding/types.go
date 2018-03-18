package encoding

import "github.com/yuuki0xff/goapptrace/tracer/types"

func MarshalSymbolsData(sd *types.SymbolsData, buf []byte) int64 {
	total := marshalStringSlice(buf, sd.Files)
	total += marshalGoModuleSlice(buf[total:], sd.Mods)
	total += marshalGoFuncSlice(buf[total:], sd.Funcs)
	total += marshalGoLineSlice(buf[total:], sd.Lines)
	return total
}
func UnmarshalSymbolsData(p *types.SymbolsData, buf []byte) int64 {
	var total int64
	var n int64
	p.Files, n = unmarshalStringSlice(buf)
	total += n
	p.Mods, n = unmarshalGoModuleSlice(buf[total:])
	total += n
	p.Funcs, n = unmarshalGoFuncSlice(buf[total:])
	total += n
	p.Lines, n = unmarshalGoLineSlice(buf[total:])
	total += n
	return total
}
func SizeSymbolsData(sd *types.SymbolsData) int64 {
	total := sizeStringSlice(sd.Files)
	total += sizeGoModuleSlice(sd.Mods)
	total += sizeGoFuncSlice(sd.Funcs)
	total += sizeGoLineSlice(sd.Lines)
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

func marshalFuncLogID(buf []byte, id types.FuncLogID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalFuncLogID(buf []byte) (types.FuncLogID, int64) {
	val, n := unmarshalUint64(buf)
	return types.FuncLogID(val), n
}

func marshalRawFuncLogID(buf []byte, id types.RawFuncLogID) int64 {
	return marshalUint64(buf, uint64(id))
}
func unmarshalRawFuncLogID(buf []byte) (types.RawFuncLogID, int64) {
	val, n := unmarshalUint64(buf)
	return types.RawFuncLogID(val), n
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
