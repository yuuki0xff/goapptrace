package types

import "strconv"

func marshalInt64(val int64) ([]byte, error) {
	s := strconv.FormatInt(val, 10)
	return []byte(s), nil
}
func unmarshalInt64(data []byte) (int64, error) {
	return strconv.ParseInt(string(data), 10, 64)
}
