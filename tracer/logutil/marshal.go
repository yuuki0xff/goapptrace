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

func (id FuncStatusID) MarshalJSON() ([]byte, error) {
	return marshalUint64(uint64(id))
}
func (id *FuncStatusID) UnmarshalJSON(data []byte) error {
	val, err := unmarshalUint64(data)
	*id = FuncStatusID(val)
	return err
}

func (t Time) MarshalJSON() ([]byte, error) {
	return marshalInt64(int64(t))
}
func (t *Time) UnmarshalJSON(data []byte) error {
	val, err := unmarshalInt64(data)
	*t = Time(val)
	return err
}

func marshalInt64(val int64) ([]byte, error) {
	s := strconv.FormatInt(val, 10)
	return []byte(s), nil
}
func unmarshalInt64(data []byte) (int64, error) {
	return strconv.ParseInt(string(data), 10, 64)
}

func marshalUint64(val uint64) ([]byte, error) {
	s := strconv.FormatUint(val, 10)
	return []byte(s), nil
}
func unmarshalUint64(data []byte) (uint64, error) {
	return strconv.ParseUint(string(data), 10, 64)
}
