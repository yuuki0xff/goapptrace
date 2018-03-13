package types

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
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

// TODO: 使用されていない?
type FuncID uint64
type GoLineID uint64

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
func (f *FuncID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncID(id)
	return nil
}

func (f *GoLineID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = GoLineID(id)
	return nil
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

func NewTxID() TxID {
	// TODO: randを使うのと、atomic.AddInt32を使うの、どちらが早いのか？
	return TxID(rand.Int63())
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
