package util

import (
	"fmt"
	"io"
	"log"

	"github.com/pkg/errors"
)

var (
	ErrPartialWrite = errors.New("partial write")
	ErrPartialRead  = errors.New("partial read error")
)

// PanicHandler handles panic and returns a error.
// If fn() does not panic, PanicHandler returns nil.
// Otherwise, PanicHandler returns an error object.
func PanicHandler(fn func()) (err error) {
	defer func() {
		if obj := recover(); obj != nil {
			var ok bool
			err, ok = obj.(error)
			if !ok {
				// convert the obj from unknown type to error type.
				err = errors.New(fmt.Sprint(obj))
			}
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
		log.Panic(ErrPartialWrite)
	}
}
func MustRead(r io.Reader, data []byte) {
	n, err := r.Read(data)
	if err != nil {
		log.Panic(err)
	}
	if n != len(data) {
		log.Panic(ErrPartialRead)
	}
}
