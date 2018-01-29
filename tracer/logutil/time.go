package logutil

import (
	"time"
)

func NewTime(t time.Time) Time {
	return Time(t.UnixNano())
}

func (t Time) UnixTime() time.Time {
	sec := int64(t) / 1e9
	nanosec := int64(t) % 1e9
	return time.Unix(sec, nanosec)
}
