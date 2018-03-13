package simulator

import (
	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// RawFuncLogから実行時の状態を推測し、FuncLogとGoroutineオブジェクトを構築する。
// 具体的には、関数やgoroutineの開始・終了のタイミングの推測を行う。
// 仕様上、監視対象外のコードで生成されたgoroutineの終了タイミングは正確でない。
// 一度終了したと判定したgoroutineが、後になってまた動いていると判定されることがある。
type StateSimulator struct {
	// 次に追加するFuncLogのID
	nextID types.FuncLogID
	// 実行中か実行が終了した関数についてのログ
	funcLogs map[types.FuncLogID]*types.FuncLog
	// RawFuncLog.TxIDに対応するFuncLogIDを保持する。
	// 関数の実行が終了したら、そのTxIDを削除すること。
	txids map[types.TxID]types.FuncLogID
	// goroutineごとの、スタックトップのFuncLogID
	// キーの存在チェックを行っていないため、goroutineの実行終了後も削除してはならない。
	stacks map[types.GID]types.FuncLogID
	// 実行中か実行が終わったgoroutine
	// 実行終了したと判断したgoroutineを動作中に変更することがあるので、
	// 実行が終了しても削除してはならない。
	goroutines map[types.GID]*types.Goroutine

	lock sync.RWMutex
}

type StateSimulatorStore struct {
	lock sync.Mutex
	m    map[string]*StateSimulator
}
