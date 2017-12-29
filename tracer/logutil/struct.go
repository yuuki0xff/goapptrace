package logutil

const (
	NotEnded      = -1
	TimeRangeStep = 5000

	FuncStart = TagName("funcStart")
	FuncEnd   = TagName("funcEnd")
)

type GID int64 // GID - Goroutine ID
type TxID uint64
type Time int
type TagName string
type GoroutineMap struct {
	m map[GID]*Goroutine
}

// RawFuncLogから実行時の状態を推測し、FuncLogとGoroutineオブジェクトを構築する。
// 具体的には、関数やgoroutineの開始・終了のタイミングの推測を行う。
// 仕様上、監視対象外のコードで生成されたgoroutineの終了タイミングは正確でない。
// 一度終了したと判定したgoroutineが、後になってまた動いていると判定されることがある。
type StateSimulator struct {
	Symbols *Symbols

	funcLogs []*FuncLog
	// トレース開始から現在までに存在していた全てのgoroutine
	goroutineMap *GoroutineMap

	// goroutine別の、現在のスタックの状態。
	// ログから推測しているので、実際の状態とは異なるかもしれない。
	stacks map[GID][]*FuncLog
}

// Goroutineの生存期間、およびそのGoroutine内で行われたアクションを保持する。
type Goroutine struct {
	GID       GID
	FuncLogs  []*FuncLog
	StartTime Time
	EndTime   Time
}

// 1回の関数呼び出しに関する情報。
// 関数の生存期間、呼び出し元の関数など
type FuncLog struct {
	StartTime Time
	EndTime   Time
	Parent    *FuncLog

	Frames []FuncStatusID
	GID    GID
}

type RawFuncLog struct {
	Time      Time
	Tag       TagName        `json:"tag"`
	Timestamp int64          `json:"timestamp"`
	Frames    []FuncStatusID `json:"frames"` // Frames[0] is current frame, Frames[1] is the caller of Frame[0].
	GID       GID            `json:"gid"`
	TxID      TxID           `json:"txid"`
}

////////////////////////////////////////////////////////////////
// Symbols
type FuncID uint64
type FuncStatusID uint64

type Symbols struct {
	Funcs      []*FuncSymbol
	FuncStatus []*FuncStatus
}

type FuncSymbol struct {
	ID    FuncID
	Name  string  // example: "github.com/yuuki0xff/goapptrace.main"
	File  string  // example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
	Entry uintptr // entry point of function
}

type FuncStatus struct {
	ID   FuncStatusID
	Func FuncID
	Line uint64
	PC   uintptr
}

type SymbolsEditor struct {
	KeepID     bool
	symbols    *Symbols
	funcs      map[string]FuncID
	funcStatus map[FuncStatus]FuncStatusID // FuncStatus.IDは常に0
}
