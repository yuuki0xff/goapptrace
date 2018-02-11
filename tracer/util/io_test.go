package util

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	errExample = errors.New("example error")
	errWrite   = errors.New("something happened while writing")
	errRead    = errors.New("something happened while reading")
	testData   = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
)

func TestPanicHandler_noPanic(t *testing.T) {
	a := assert.New(t)
	a.NoError(PanicHandler(func() {}))
}
func TestPanicHandler_panicWithError(t *testing.T) {
	a := assert.New(t)
	a.EqualError(PanicHandler(func() {
		panic(errExample)
	}), errExample.Error())
}
func TestPanicHandler_panicWithString(t *testing.T) {
	a := assert.New(t)
	a.EqualError(PanicHandler(func() {
		panic(errExample.Error())
	}), errExample.Error())
}

func TestMustWrite_good(t *testing.T) {
	var gw GoodRW
	a := assert.New(t)
	a.NotPanics(func() {
		MustWrite(gw, testData)
	})
}
func TestMustWrite_error(t *testing.T) {
	var ew ErrorRW
	a := assert.New(t)
	a.PanicsWithValue(errWrite.Error(), func() {
		MustWrite(ew, testData)
	})
}
func TestMustWrite_partial(t *testing.T) {
	var pw PartialRW
	a := assert.New(t)
	a.PanicsWithValue(ErrPartialWrite.Error(), func() {
		MustWrite(pw, testData)
	})
}

func TestMustRead_good(t *testing.T) {
	var gr GoodRW
	a := assert.New(t)
	a.NotPanics(func() {
		buf := make([]byte, len(testData))
		MustRead(gr, buf)
	})
}
func TestMustRead_error(t *testing.T) {
	var er ErrorRW
	a := assert.New(t)
	a.PanicsWithValue(errRead.Error(), func() {
		buf := make([]byte, len(testData))
		MustRead(er, buf)
	})
}
func TestMustRead_partial(t *testing.T) {
	var pr PartialRW
	a := assert.New(t)
	a.PanicsWithValue(ErrPartialRead.Error(), func() {
		buf := make([]byte, len(testData))
		MustRead(pr, buf)
	})
}

type GoodRW struct{}

func (GoodRW) Read(p []byte) (n int, err error) {
	return len(p), nil
}
func (GoodRW) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type ErrorRW struct{}

func (w ErrorRW) Read(p []byte) (n int, err error) {
	return 0, errRead
}
func (w ErrorRW) Write(p []byte) (n int, err error) {
	return 0, errWrite
}

type PartialRW struct{}

func (PartialRW) Read(p []byte) (n int, err error) {
	return len(p) - 1, nil
}
func (PartialRW) Write(p []byte) (n int, err error) {
	return len(p) - 1, nil
}
