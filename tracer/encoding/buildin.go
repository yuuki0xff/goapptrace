package encoding

import (
	"encoding/binary"
)

func marshalBool(buf []byte, val bool) int64 { // nolint
	if val {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	return 1
}
func unmarshalBool(buf []byte) (bool, int64) { // nolint
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
func MarshalUint8(buf []byte, val uint8) int64 {
	buf[0] = val
	return 1
}
func UnmarshalUint8(buf []byte) (uint8, int64) {
	return buf[0], 1
}

func MarshalString(buf []byte, str string) int64 {
	total := marshalUint64(buf, uint64(len(str)))
	total += int64(copy(buf[total:], []byte(str)))
	return total
}
func UnmarshalString(buf []byte) (string, int64) {
	length, n := unmarshalUint64(buf)
	buf = buf[n:]
	return string(buf[:length]), n + int64(length)
}
func sizeString(str string) int64 {
	return int64(8 + len(str))
}

func marshalStringSlice(buf []byte, strs []string) int64 {
	var total int64

	n := marshalUint64(buf[total:], uint64(len(strs)))
	total += n
	for i := range strs {
		n = MarshalString(buf[total:], strs[i])
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
		strs[i], n = UnmarshalString(buf[total:])
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

//go:nosplit
func MarshalUintptr(buf []byte, ptr uintptr) int64 {
	return marshalUint64(buf, uint64(ptr))
}

//go:nosplit
func UnmarshalUintptr(buf []byte) (uintptr, int64) {
	ptr, n := unmarshalUint64(buf)
	return uintptr(ptr), n
}

func marshalUintptrSlice(buf []byte, slice []uintptr) int64 {
	total := marshalUint64(buf, uint64(len(slice)))
	for i := range slice {
		total += marshalUint64(buf[total:], uint64(slice[i]))
	}
	return total
}
func unmarshalUintptrSlice(buf []byte, slicep *[]uintptr) int64 {
	var total int64
	length, n := unmarshalUint64(buf)
	total += n

	*slicep = (*slicep)[:length]
	s := *slicep
	for i := range s {
		ptr, n := unmarshalUint64(buf[total:])
		s[i] = uintptr(ptr)
		total += n
	}
	return total
}
