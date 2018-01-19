package logutil

import (
	"strconv"
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

func (t Time) MarshalText() (text []byte, err error) {
	s := strconv.FormatInt(int64(t), 10)
	return []byte(s), nil
}
func (t *Time) UnmarshalText(text []byte) error {
	i, err := strconv.ParseInt(string(text), 10, 64)
	if err != nil {
		return err
	}
	*t = Time(i)
	return nil
}
