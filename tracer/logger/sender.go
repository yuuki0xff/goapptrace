package logger

import "github.com/yuuki0xff/goapptrace/tracer/logutil"

// Sender is interface for send or store of logs.
type Sender interface {
	Open() error
	Close() error
	Send(diff *logutil.SymbolsDiff, raw *logutil.RawFuncLog) error
}
