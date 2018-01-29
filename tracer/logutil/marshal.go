package logutil

import "encoding/json"

// 他のパッケージで定義されたuint64ベースの型をUnmarshalしようとすると、このようなエラーが発生する。
// このエラーを回避するためのメソッドを定義している。
//
//   "json: cannot unmarshal number into Go value of type xxxx xxxx"

func (id FuncID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint64(id))
}
func (id *FuncID) UnmarshalJSON(data []byte) error {
	var val uint64
	err := json.Unmarshal(data, &val)
	*id = FuncID(val)
	return err
}

func (id FuncStatusID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint64(id))
}
func (id *FuncStatusID) UnmarshalJSON(data []byte) error {
	var val uint64
	err := json.Unmarshal(data, &val)
	*id = FuncStatusID(val)
	return err
}
