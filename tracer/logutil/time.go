package logutil

import (
	"strconv"
	"time"
)

func NewTime(t time.Time) Time {
	return Time(t.Unix())
}

func (t Time) UnixTime() time.Time {
	return time.Unix(int64(t), 0)
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
