package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"encoding/json"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// FileSender writes Symbols and FuncLog to log file.
type FileSender struct {
	file io.WriteCloser
}

// open log file.
func (f *FileSender) Open() error {
	var err error
	f.file, err = os.OpenFile(f.logFilePath(), os.O_CREATE|os.O_WRONLY, 0644)
	return err
}

// close log file.
// これ移行はSendできない。
func (f *FileSender) Close() error {
	if err := f.file.Close(); err != nil {
		return err
	}
	f.file = nil
	return nil
}

// write Symbols and RawFuncLog to the log file.
func (f *FileSender) Send(symbols *logutil.Symbols, funclog *logutil.RawFuncLogNew) error {
	enc := json.NewEncoder(f.file)
	// write symbols to file
	if symbols != nil {
		err := enc.Encode(symbols)
		if err != nil {
			return err
		}
	}

	// write backtrace to file
	err := enc.Encode(funclog)
	if err != nil {
		return err
	}
	return nil
}

// returns absolute path of log file.
func (f *FileSender) logFilePath() string {
	pid := os.Getpid()
	prefix, ok := os.LookupEnv(info.DEFAULT_LOGFILE_ENV)
	if !ok {
		prefix = info.DEFAULT_LOGFILE_PREFIX
	}
	relativePath := fmt.Sprintf("%s.%d.log", prefix, pid)
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		log.Panic(err)
	}
	return absPath
}
