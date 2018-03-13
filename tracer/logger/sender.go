package logger

import "github.com/yuuki0xff/goapptrace/tracer/types"

// Sender is interface for send or store of logs.
type Sender interface {
	Open() error
	Close() error
	// SymbolsDataをサーバに送信する。
	// この関数の実行終了後はdataの変更や破棄をしても構わない。
	// SendSymbols()は、関数の実行終了までにdataが変更されても構わない状態にしなければならない。
	SendSymbols(data *types.SymbolsData) error
	// RawFuncLogをサーバに送信する。
	// この関数の実行終了後はrawの変更や破棄をしても構わない。
	// SendLog()は、関数の実行終了までにrawが変更されても構わない状態にしなければならない。
	SendLog(raw *types.RawFuncLog) error
}
