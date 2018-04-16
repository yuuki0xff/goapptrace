package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// FileSender writes Symbols and FuncLog to log file.
type FileSender struct {
	w *storage.CompactLogWriter
}

// open log file.
func (f *FileSender) Open() error {
	clog := storage.CompactLog{
		File: storage.File(f.logFilePath()),
	}
	f.w = clog.Writer()
	return f.w.Open()
}

// close log file.
// これ移行はSendできない。
func (f *FileSender) Close() error {
	if f.w == nil {
		return ClosedError
	}
	err := f.w.Close()
	f.w = nil
	return err
}

// write Symbols to the log file.
func (f *FileSender) SendSymbols(data *types.SymbolsData) error {
	if f.w == nil {
		return ClosedError
	}
	return f.w.Write(data, nil)
}

// write RawFuncLog to the log file.
func (f *FileSender) SendLog(raw *types.RawFuncLog) error {
	if f.w == nil {
		return ClosedError
	}
	return f.w.Write(nil, raw)
}

// returns absolute path of log file.
func (f *FileSender) logFilePath() string {
	pid := os.Getpid()
	prefix, ok := os.LookupEnv(info.DefaultLogfileEnv)
	if !ok {
		prefix = info.DefaultLogfilePrefix
	}
	relativePath := fmt.Sprintf("%s.%d.log.gz", prefix, pid)
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		log.Panic(err)
	}
	return absPath
}
