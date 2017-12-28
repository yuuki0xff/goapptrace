package logutil

const (
	NotEnded      = -1
	TimeRangeStep = 5000

	FuncStart = TagName("funcStart")
	FuncEnd   = TagName("funcEnd")
)

type LoadRawLogHandler func(*RawFuncLog)
type LoadFuncLogHandler func(*FuncLog)
type GID int64 // GID - Goroutine ID
type TxID uint64
type Time int
type TagName string
type TimeRange struct{ rangeID int }
type RecordList []*FuncLog
type GoroutineMap struct {
	m map[GID]*Goroutine
}

// 生存しているGoroutineを時間帯別に集計して保持する。
// 粒度が小さくなってしまうものの、同等の機能は storage.IndexRecord にフィールドを追加することで可能。
type TimeRangeMap struct {
	m map[TimeRange]*GoroutineMap
}

// TODO: 要リファクタ
type StateSimulator struct {
	Symbols *Symbols

	////////////////
	// ↓ 初期化不要 ↓

	// 関数の生存期間を記録したレコードのリスト。
	Records RecordList
	// トレース開始から現在までに存在していた全てのgoroutine
	GoroutineMap *GoroutineMap

	// goroutine別の、現在のスタックの状態。
	// ログから推測しているので、実際の状態とは異なるかもしれない。
	gmap map[GID][]*FuncLog
}

// Goroutineの生存期間、およびそのGoroutine内で行われたアクションを保持する。
type Goroutine struct {
	GID       GID
	Records   RecordList
	StartTime Time
	EndTime   Time
}

// 関数の生存期間、呼び出し元の関数のログなど
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
