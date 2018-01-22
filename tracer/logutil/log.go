package logutil

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

// LogIDを16進数表現で返す。
func (id LogID) Hex() string {
	return hex.EncodeToString(id[:])
}

// 16新数表現からLogIDに変換して返す。
// 16次sン数でない文字列や、長さが一致しない文字列が与えられた場合はエラーを返す。
func (LogID) Unhex(str string) (id LogID, err error) {
	var buf []byte
	buf, err = hex.DecodeString(str)
	if err != nil {
		return
	}
	if len(buf) != len(id) {
		err = fmt.Errorf(
			"missmatch id length. expect %d charactors, but %d",
			2*len(id), 2*len(buf),
		)
		return
	}
	copy(id[:], buf)
	return
}

// 16進数化した文字列として出力する。
func (id LogID) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.Write([]byte(id.Hex()))
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

// 16進数値のような文字列からLogIDに変換する。
func (id *LogID) UnmarshalJSON(data []byte) error {
	if len(data) != len(id)*2+2 {
		return errors.New("mismatch id length")
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("missing '\"'")
	}

	newId, err := id.Unhex(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*id = newId
	return nil
}

// LogIDを16進数表現で返す。
func (id LogID) String() string {
	return id.Hex()
}
