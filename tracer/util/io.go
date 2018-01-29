package util

import (
	"io"
	"log"

	"github.com/pkg/errors"
)

func PanicHandler(fn func()) (err error) {
	defer func() {
		if obj := recover(); obj != nil {
			err = obj.(error)
		}
	}()
	fn()
	return nil
}

func MustWrite(w io.Writer, data []byte) {
	n, err := w.Write(data)
	if err != nil {
		log.Panic(err)
	}
	if n != len(data) {
		log.Panic(errors.New("partial write error"))
	}
}
func MustRead(r io.Reader, data []byte) {
	n, err := r.Read(data)
	if err != nil {
		log.Panic(err)
	}
	if n != len(data) {
		log.Panic(errors.New("partial read error"))
	}
}
