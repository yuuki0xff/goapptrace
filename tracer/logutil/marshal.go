package logutil

import (
	"strconv"
)

// 他のパッケージで定義されたuint64ベースの型をUnmarshalしようとすると、このようなエラーが発生する。
// このエラーを回避するためのメソッドを定義している。
//
//   "json: cannot unmarshal number into Go value of type xxxx xxxx"

func (id FuncID) MarshalJSON() ([]byte, error) {
	return marshalUint64(uint64(id))
}
func (id *FuncID) UnmarshalJSON(data []byte) error {
	val, err := unmarshalUint64(data)
	*id = FuncID(val)
	return err
}

func (id GoLineID) MarshalJSON() ([]byte, error) {
	return marshalUint64(uint64(id))
}
func (id *GoLineID) UnmarshalJSON(data []byte) error {
	val, err := unmarshalUint64(data)
	*id = GoLineID(val)
	return err
}

func marshalUint64(val uint64) ([]byte, error) {
	s := strconv.FormatUint(val, 10)
	return []byte(s), nil
}
func unmarshalUint64(data []byte) (uint64, error) {
	return strconv.ParseUint(string(data), 10, 64)
}
