package schema

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/yuuki0xff/goapptrace/config"
)

const (
	NotEnded       = Time(-1)
	NotFoundParent = FuncLogID(-1)
)
const (
	FuncStart TagName = iota
	FuncEnd
)

type GID int64 // GID - Goroutine ID
type TxID uint64
type FuncLogID int
type RawFuncLogID int
type Time int64
type TagName uint8
type LogID [16]byte

func (gid GID) String() string {
	return strconv.FormatInt(int64(gid), 10)
}

func NewTime(t time.Time) Time {
	return Time(t.UnixNano())
}
func (t Time) String() string {
	return t.UnixTime().Format(config.TimestampFormat)
}
func (t Time) NumberString() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t Time) UnixTime() time.Time {
	sec := int64(t) / 1e9
	nanosec := int64(t) % 1e9
	return time.Unix(sec, nanosec)
}
func (t Time) MarshalJSON() ([]byte, error) {
	return marshalInt64(int64(t))
}
func (t *Time) UnmarshalJSON(data []byte) error {
	val, err := unmarshalInt64(data)
	*t = Time(val)
	return err
}

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
