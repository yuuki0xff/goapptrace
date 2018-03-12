package logutil

import (
	"strconv"
	"time"

	"github.com/yuuki0xff/goapptrace/config"
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
