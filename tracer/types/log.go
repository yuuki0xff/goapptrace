package types

import "sync"

const MaxStackSize = 64

// Goroutineの生存期間、およびそのGoroutine内で行われたアクションを保持する。
// 実行終了後も、変更されることがある。
type Goroutine struct {
	GID       GID  `json:"goroutine-id"`
	StartTime Time `json:"start-time"`
	EndTime   Time `json:"end-time"`
}

// 1回の関数呼び出しに関する情報。
// 関数の生存期間、呼び出し元の関数など。
// 関数の実行終了後は、フィールドの値は変更されない。
type FuncLog struct {
	ID        FuncLogID `json:"id"`
	StartTime Time      `json:"start-time"`
	EndTime   Time      `json:"end-time"`
	ParentID  FuncLogID `json:"parent-id"`

	Frames []uintptr `json:"frames"`
	GID    GID       `json:"gid"`
}

type RawFuncLog struct {
	ID        RawFuncLogID
	Tag       TagName `json:"tag"`
	Timestamp Time    `json:"timestamp"`
	// 関数呼び出し時のスタックトレース。
	// Frames を外部で保持したい場合、新しいスライスを確保し、中身をコピーすること。
	// RawFuncLog は再利用されるため、スライスのコピーを行うと意図しないタイミングで Frames の内容が破壊される可能性がある。
	Frames []uintptr `json:"frames"` // Frames[0] is current frame, Frames[1] is the caller of Frame[0].
	GID    GID       `json:"gid"`
	TxID   TxID      `json:"txid"`
}

func (fl FuncLog) IsEnded() bool {
	return fl.EndTime != NotEnded
}

// RawFuncLog オブジェクトが再利用できるように蓄えておく。
// メモリ確保の回数が減るため、パフォーマンス向上が期待できる。
//
// 取得したオブジェクトの RawFuncLog.Frames スライスのキャパシティは types.MaxStackSize である。
// RawFuncLogPool.Get() をしたら、 RawFuncLog.Frames の長さを拡張することを忘れずに。
// RawFuncLogPool.Put() する場合、 RawFuncLog.Frames フィールドは nil にしてはならない。また、他のスライスを代入することもしてはいけない。
//
// Example:
//   logmsg := types.RawFuncLogPool.Get().(*types.RawFuncLog)
//   logmsg.Frames = logmsg.Frames[:cap(logmsg.Frames)
//   // do something
//   types.RawFuncLogPool.Put(logmsg)
var RawFuncLogPool = sync.Pool{
	New: func() interface{} {
		return &RawFuncLog{
			Frames: FramesPool.Get().([]uintptr),
		}
	},
}

var FuncLogPool = sync.Pool{
	New: func() interface{} {
		return &FuncLog{
			Frames: FramesPool.Get().([]uintptr),
		}
	},
}

var FramesPool = sync.Pool{
	New: func() interface{} {
		return make([]uintptr, MaxStackSize)
	},
}
