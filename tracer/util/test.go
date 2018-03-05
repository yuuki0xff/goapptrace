package util

import (
	"io/ioutil"
	"os"
)

// WithTempFile create a temporary file and calls fn with file path.
func WithTempFile(fn func(tmpfile string)) {
	file, err := ioutil.TempFile("", ".goapptrace.test")
	if err != nil {
		panic(err)
	}
	defer func() {
		err = os.Remove(file.Name())
		if err != nil {
			panic(err)
		}
	}()

	fn(file.Name())
}

// WithTempDir create a temporary directory and chdir into it.
func WithTempDir(fn func()) {
	dir, err := ioutil.TempDir("", ".goapptrace.test")
	if err != nil {
		panic(err)
	}
	defer func() {
		err = os.Chdir("/")
		if err != nil {
			panic(err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			panic(err)
		}
	}()

	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}

	fn()
}
