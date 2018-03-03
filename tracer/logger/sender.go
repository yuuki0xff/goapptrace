package logger

import "github.com/yuuki0xff/goapptrace/tracer/logutil"

// Sender is interface for send or store of logs.
type Sender interface {
	Open() error
	Close() error
	Send(diff *logutil.SymbolsData, raw *logutil.RawFuncLog) error
}
