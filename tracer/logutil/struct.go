package logutil

import (
	"sync"
)

const (
	NotEnded       = Time(-1)
	NotFoundParent = FuncLogID(-1)

	FuncStart = TagName("funcStart")
	FuncEnd   = TagName("funcEnd")
)

type GID int64 // GID - Goroutine ID
type TxID uint64
type FuncLogID int
type RawFuncLogID int
type Time int64
type TagName string
type LogID [16]byte

// RawFuncLogから実行時の状態を推測し、FuncLogとGoroutineオブジェクトを構築する。
// 具体的には、関数やgoroutineの開始・終了のタイミングの推測を行う。
// 仕様上、監視対象外のコードで生成されたgoroutineの終了タイミングは正確でない。
// 一度終了したと判定したgoroutineが、後になってまた動いていると判定されることがある。
type StateSimulator struct {
	// 次に追加するFuncLogのID
	nextID FuncLogID
	// 実行中か実行が終了した関数についてのログ
	funcLogs map[FuncLogID]*FuncLog
	// RawFuncLog.TxIDに対応するFuncLogIDを保持する。
	// 関数の実行が終了したら、そのTxIDを削除すること。
	txids map[TxID]FuncLogID
	// goroutineごとの、スタックトップのFuncLogID
	// キーの存在チェックを行っていないため、goroutineの実行終了後も削除してはならない。
	stacks map[GID]FuncLogID
	// 実行中か実行が終わったgoroutine
	// 実行終了したと判断したgoroutineを動作中に変更することがあるので、
	// 実行が終了しても削除してはならない。
	goroutines map[GID]*Goroutine

	lock sync.RWMutex
}

type StateSimulatorStore struct {
	lock sync.Mutex
	m    map[string]*StateSimulator
}

// Goroutineの生存期間、およびそのGoroutine内で行われたアクションを保持する。
// 実行終了後も、変更されることがある。
type Goroutine struct {
	GID       GID
	StartTime Time
	EndTime   Time
}

// 1回の関数呼び出しに関する情報。
// 関数の生存期間、呼び出し元の関数など。
// 関数の実行終了後は、フィールドの値は変更されない。
type FuncLog struct {
	ID        FuncLogID
	StartTime Time
	EndTime   Time
	ParentID  FuncLogID

	Frames []GoLineID
	GID    GID
}

type RawFuncLog struct {
	// TODO: ID fieldに適切な値を書き込む
	// TODO: ドキュメントを書く

	ID        RawFuncLogID
	Tag       TagName    `json:"tag"`
	Timestamp Time       `json:"timestamp"`
	Frames    []GoLineID `json:"frames"` // Frames[0] is current frame, Frames[1] is the caller of Frame[0].
	GID       GID        `json:"gid"`
	TxID      TxID       `json:"txid"`
}

func (fl FuncLog) IsEnded() bool {
	return fl.EndTime != NotEnded
}

////////////////////////////////////////////////////////////////
// Symbols
type FuncID uint64
type GoLineID uint64

type SymbolsReadFn func() (SymbolsData, error)
type SymbolsWriteFn func(data SymbolsData) error

type Symbols struct {
	Writable bool
	// KeepIDがtrueのとき、FuncIDおよびGoLineIDは、追加時に指定されたIDを使用する。
	// KeepIDがfalseのとき、追加時に指定されたIDは無視し、新たなIDを付与する。
	KeepID bool

	lock sync.RWMutex
	data SymbolsData
}

type SymbolsData struct {
	Funcs  []*GoFunc
	GoLine []*GoLine
}

// FileID is index of Symbols.Files array.
type FileID uint64

// File is file path to the source code.
// example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
type File string

// GoModules means a module in golang.
//type GoModule struct {
//	Name  string
//	MinPC uintptr
//	MaxPC uintptr
//}

// GoFunc means a function in golang.
//type GoFunc struct {
//	Entry uintptr
//	// example: "github.com/yuuki0xff/goapptrace.main"
//	Name string
//	FileID FileID
//}

// GoLine haves a correspondence to position on source code from PC (Program Counter).
//type GoLine struct {
//	PC     uintptr
//	FileID FileID
//	Line   uint64
//}

type GoFunc struct {
	ID    FuncID
	Name  string  // example: "github.com/yuuki0xff/goapptrace.main"
	File  string  // example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
	Entry uintptr // entry point of function
}

type GoLine struct {
	ID   GoLineID
	Func FuncID
	Line uint64
	PC   uintptr
}
