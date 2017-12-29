package logutil

const (
	NotEnded      = -1
	TimeRangeStep = 5000

	FuncStart = TagName("funcStart")
	FuncEnd   = TagName("funcEnd")
)

type GID int64 // GID - Goroutine ID
type TxID uint64
type FuncLogID int
type Time int
type TagName string

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
