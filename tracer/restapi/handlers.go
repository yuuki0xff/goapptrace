package restapi

import (
	"container/heap"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/simulator"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"golang.org/x/sync/errgroup"
)

const (
	NoSortOrder         SortOrder = ""
	AscendingSortOrder  SortOrder = "asc"
	DescendingSortOrder SortOrder = "desc"

	NoSortKey       SortKey = ""
	SortByID        SortKey = "id"
	SortByStartTime SortKey = "start-time"
	SortByEndTime   SortKey = "end-time"
)

type RouterArgs struct {
	Config         *config.Config
	Storage        *storage.Storage
	SimulatorStore *simulator.StateSimulatorStore
}

// Goapptrace REST API v0.xのハンドラを提供する
type APIv0 struct {
	RouterArgs
	Logger *log.Logger
}

// APIのレスポンスの生成を支援するworker。
// api.worker() メソッドから作成し APIWorker.wait() で待ち合わせをする。
// パイプラインモデルの並列処理を行う。
type APIWorker struct {
	Api        *APIv0
	Args       *RouterArgs
	Logger     *log.Logger
	BufferSize int
	Logobj     *storage.Log

	group *errgroup.Group
	ctx   context.Context
}

type FuncLogAPIWorker struct {
	api  *APIWorker
	inCh chan *types.FuncLog
	// 呼び出すと、readerとfilterが終了する。
	stopReader func()
	// readerとfilterに対するcontext。
	// limiterは最後までchannelを読み取らなかったときに、readerやfilterを終了させるために使用する。
	readCtx context.Context
	// sorterとlimiterに対するcontext。
	sortCtx context.Context
	// senderに対するcontext。
	sendCtx context.Context
}

type Encoder interface {
	Encode(v interface{}) error
}

// メソッドの処理を置き換え可能なheap.Interfaceの実装。
type GenericHeap struct {
	LenFn  func() int
	LessFn func(i, j int) bool
	SwapFn func(i, j int)
	PushFn func(x interface{})
	PopFn  func() interface{}
}

type HttpRequestHandler func(w http.ResponseWriter, r *http.Request)
type APIRequestHandler func(w http.ResponseWriter, r *http.Request) (status int, data interface{}, err error)

func NewRouter(args RouterArgs) *mux.Router {
	router := mux.NewRouter()

	apiv0 := APIv0{
		RouterArgs: args,
		Logger:     log.New(os.Stdout, "[REST API] ", 0),
	}
	apiv0.SetHandlers(router)
	return router
}

func (api APIv0) SetHandlers(router *mux.Router) {
	v01 := router.PathPrefix("/api/v0.1").Subrouter()
	v01.HandleFunc("/servers", api.servers).Methods(http.MethodGet)
	v01.HandleFunc("/server/{server-id}/status", api.serverStatus).Methods(http.MethodGet)
	v01.HandleFunc("/server/{server-id}/status", api.serverStatus).Methods(http.MethodPut).
		Queries("version", "{version:[0-9]+}")
	v01.HandleFunc("/server/{server-id}/watch", api.notImpl).Methods(http.MethodGet).
		Queries(
			"version", "{version:[0-9]+}",
			"timeout", "{timeout:[0-9]+}",
		)

	v01.HandleFunc("/logs", api.logs).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}", api.log).Methods(http.MethodDelete)
	v01.HandleFunc("/log/{log-id}", api.log).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}", api.log).Methods(http.MethodPut).
		Queries("version", "{version:[0-9]+}")
	v01.HandleFunc("/log/{log-id}/watch", api.notImpl).Methods(http.MethodGet).
		Queries(
			"version", "{version:[0-9]+}",
			"timeout", "{timeout:[0-9]+}",
		)

	v01.HandleFunc("/log/{log-id}/func-call/search", api.funcCallSearch).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/func-call/stream", api.notImpl).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/goroutines/search", api.goroutineSearch).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbols", api.symbols).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbol/module/{pc}", api.goModule).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbol/func/{pc}", api.goFunc).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbol/line/{pc}", api.goLine).Methods(http.MethodGet)

	v01.HandleFunc("/tracers", api.tracers).Methods(http.MethodGet)
	v01.HandleFunc("/tracer/{tracer-id}", api.tracer).Methods(http.MethodDelete)
	v01.HandleFunc("/tracer/{tracer-id}/status", api.tracers).Methods(http.MethodGet)
	v01.HandleFunc("/tracer/{tracer-id}/status", api.tracer).Methods(http.MethodPut)
	v01.HandleFunc("/tracer/{tracer-id}/watch", api.notImpl).Methods(http.MethodGet)
}
func (api APIv0) serverError(w http.ResponseWriter, err error, msg string) {
	api.Logger.Println(errors.Wrap(err, "failed to json.Marshal").Error())
	http.Error(w, msg, http.StatusInternalServerError)
}
func (api APIv0) write(w io.Writer, data []byte) {
	_, err := w.Write(data)
	if err != nil {
		api.Logger.Println(errors.Wrap(err, "failed to Write").Error())
	}
}
func (api APIv0) writeObj(w http.ResponseWriter, obj interface{}) {
	js, err := json.Marshal(obj)
	if err != nil {
		api.serverError(w, err, "failed to json.Marshal")
		return
	}
	api.write(w, js)
}

// TODO: テストを書く
func (api APIv0) servers(w http.ResponseWriter, r *http.Request) {
	srvList := make([]ServerStatus, 0, len(api.Config.Servers.LogServer))
	for _, srv := range api.Config.Servers.LogServer {
		srvList = append(srvList, ServerStatus(*srv))
	}

	js, err := json.Marshal(Servers{
		Servers: srvList,
	})
	if err != nil {
		api.serverError(w, err, "failed to json.Marshal")
		return
	}
	api.write(w, js)
}

// TODO: テストを書く
func (api APIv0) serverStatus(w http.ResponseWriter, r *http.Request) {
	strId := mux.Vars(r)["server-id"]
	id, err := strconv.Atoi(strId)
	if err != nil {
		http.Error(w, "server-id is invalid", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		srv, ok := api.Config.Servers.LogServer[config.ServerID(id)]
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		js, err := json.Marshal(srv)
		if err != nil {
			api.serverError(w, err, "failed to json.Marshal")
			return
		}
		api.write(w, js)
	case http.MethodPut:
		// TODO: impl
		api.notImpl(w, r)
	}
}

// TODO: テストを書く
func (api APIv0) logs(w http.ResponseWriter, r *http.Request) {
	var res Logs
	logs, err := api.Storage.Logs()
	if err != nil {
		api.serverError(w, err, "failed to load logs from storage")
		return
	}

	for _, l := range logs {
		res.Logs = append(res.Logs, l.LogInfo())
	}

	js, err := json.Marshal(res)
	if err != nil {
		api.serverError(w, err, "failed to json.Marshal")
		return
	}
	api.write(w, js)
}

// TODO: テストを書く
func (api APIv0) log(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodDelete:
		err := logobj.Remove()
		if err != nil {
			api.serverError(w, err, "failed to remove a log")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		js, err := json.Marshal(logobj.Metadata)
		if err != nil {
			api.serverError(w, err, "failed to json.Marshal()")
			return
		}
		api.write(w, js)
	case http.MethodPut:
		js, err := ioutil.ReadAll(r.Body)
		if err != nil {
			api.serverError(w, err, "failed to read from request body")
			return
		}

		meta := &types.LogMetadata{}
		if err = json.Unmarshal(js, meta); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		currentVer, err := strconv.Atoi(r.URL.Query().Get("version"))
		if err != nil {
			http.Error(w, "invalid version number", http.StatusBadRequest)
			return
		}

		if err = logobj.UpdateMetadata(currentVer, meta); err != nil {
			if err == storage.ErrConflict {
				// バージョン番号が異なるため、Metadataを更新できない。
				// 現在の状態を返す。
				js, err = json.Marshal(logobj.Metadata)
				if err != nil {
					api.serverError(w, err, "failed to json.Marshal()")
					return
				}
				w.WriteHeader(http.StatusConflict)
				api.write(w, js)
				return
			} else {
				// よく分からんエラー
				api.serverError(w, err, "failed to Log.UpdateMetadata")
				return
			}
		}

		// 更新に成功。
		// 新しい状態を返す。
		js, err = json.Marshal(logobj.Metadata)
		if err != nil {
			api.serverError(w, err, "failed to json.Marshal()")
			return
		}
		api.write(w, js)
	}
}

// TODO: テストを書く
func (api APIv0) funcCallSearch(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	var gid int64
	var minId int64
	var maxId int64
	var minTs types.Time
	var maxTs types.Time
	var limit int64
	var sortKey SortKey
	var order SortOrder
	var err error

	gid, err = parseInt(q.Get("gid"), -1)
	if err != nil {
		http.Error(w, "invaid gid", http.StatusBadRequest)
		return
	}
	minId, err = parseInt(q.Get("min-id"), -1)
	if err != nil {
		http.Error(w, "invalid min-id", http.StatusBadRequest)
		return
	}
	maxId, err = parseInt(q.Get("max-id"), -1)
	if err != nil {
		http.Error(w, "invalid max-id", http.StatusBadRequest)
		return
	}
	minTs, err = parseTimestamp(q.Get("min-timestamp"), -1)
	if err != nil {
		http.Error(w, "invalid min-timestamp", http.StatusBadRequest)
		return
	}
	maxTs, err = parseTimestamp(q.Get("max-timestamp"), -1)
	if err != nil {
		http.Error(w, "invalid max-timestamp", http.StatusBadRequest)
		return
	}
	limit, err = parseInt(q.Get("limit"), -1)
	if err != nil {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return
	}
	sortKey, err = parseSortKey(q.Get("sort"))
	if err != nil {
		http.Error(w, "invalid sort", http.StatusBadRequest)
		return
	}
	order, err = parseOrder(q.Get("order"), AscendingSortOrder)
	if err != nil {
		http.Error(w, "invalid order", http.StatusBadRequest)
		return
	}

	var sortFn func(f1, f2 *types.FuncLog) bool
	switch sortKey {
	case SortByID:
		sortFn = func(f1, f2 *types.FuncLog) bool {
			return f1.ID < f2.ID
		}
	case SortByStartTime:
		sortFn = func(f1, f2 *types.FuncLog) bool {
			return f1.StartTime < f2.StartTime
		}
	case SortByEndTime:
		sortFn = func(f1, f2 *types.FuncLog) bool {
			return f1.EndTime < f2.EndTime
		}
	case NoSortKey:
	default:
		log.Panic("bug")
	}

	switch order {
	case AscendingSortOrder:
		// 何もしない
	case DescendingSortOrder:
		// 降順にするために、大小を入れ替える。
		oldSortFn := sortFn
		sortFn = func(f1, f2 *types.FuncLog) bool {
			return oldSortFn(f2, f1)
		}
	default:
		log.Panic("bug")
	}

	// narrow the search range by ID and Timestamp.
	if minId >= 0 || maxId >= 0 || minTs >= 0 || maxTs >= 0 {
		if minId < 0 {
			minId = 0
		}
		if maxId < 0 {
			maxId = math.MaxInt64
		}

		found := false
		newMinId := int64(math.MaxInt64)
		newMaxId := int64(math.MinInt64)

		logobj.Index(func(index *storage.Index) {
			for i := int64(0); i < index.Len(); i++ {
				ir := index.Get(i)
				if !ir.IsOverlapID(minId, maxId) || !ir.IsOverlapTime(minTs, maxTs) {
					continue
				}

				found = true
				if ir.MinID < newMinId {
					newMinId = ir.MinID
				}
				if newMaxId < ir.MaxID {
					newMaxId = ir.MaxID
				}
			}
		})

		if !found {
			// 一致するレコードが存在しないため、このまま終了する
			return
		}
		if newMinId < minId || maxId < newMaxId {
			log.Panic(fmt.Errorf("assertion error: newMinId=%d < minId=%d || maxId=%d < newMaxId=%d", newMinId, minId, maxId, newMaxId))
		}
		minId = newMinId
		maxId = newMaxId
	}

	// evtが除外されるべきレコードなら、trueを返す。
	isFiltered := func(evt *types.FuncLog) bool {
		if gid >= 0 && evt.GID != types.GID(gid) {
			return true
		}
		if minId >= 0 && evt.ID < types.FuncLogID(minId) {
			return true
		}
		if maxId >= 0 && types.FuncLogID(maxId) < evt.ID {
			return true
		}
		if minTs >= 0 && (evt.StartTime < minTs && evt.EndTime < minTs) {
			return true
		}
		if maxTs >= 0 && maxTs < evt.StartTime {
			return true
		}
		return false
	}

	enc := json.NewEncoder(w)

	parentCtx := context.Background()
	worker := api.worker(parentCtx, logobj)
	fw := worker.readFuncLog(minId, maxId)
	fw = fw.filterFuncLog(isFiltered)
	fw = fw.sortAndLimit(sortFn, limit)
	fw.sendTo(enc)

	if err = worker.wait(); err != nil {
		log.Println(errors.Wrap(err, "funcCallSearch:"))
	}
}

// TODO: テストを書く
func (api APIv0) goroutineSearch(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	minTs, err := parseTimestamp(q.Get("min-timestamp"), -1)
	if err != nil {
		http.Error(w, "invalid min-timestamp", http.StatusBadRequest)
		return
	}
	maxTs, err := parseTimestamp(q.Get("max-timestamp"), -1)
	if err != nil {
		http.Error(w, "invalid max-timestamp", http.StatusBadRequest)
		return
	}

	// read all records in the search range.
	ch := make(chan types.Goroutine, 1<<20) // buffer size is 1M records
	go func() {
		defer close(ch)
		var err error
		logobj.Goroutine(func(store *storage.GoroutineStore) {
			n := store.Records()
			for i := int64(0); i < n; i++ {
				var g types.Goroutine
				err = store.GetNolock(types.GID(i), &g)
				if err != nil {
					return
				}

				if (minTs == -1 || minTs <= g.StartTime) && (maxTs == -1 || g.EndTime <= maxTs) {
					ch <- g
				}
			}
		})
		if err != nil {
			api.Logger.Println(errors.Wrap(err, "failed to read GoroutineFile"))
			return
		}
	}()

	// encode and send records to client.
	enc := json.NewEncoder(w)
	for g := range ch {
		if err := enc.Encode(g); err != nil {
			api.Logger.Println(errors.Wrap(err, "failed to json.Encoder.Encode()"))
			return
		}
	}
}
func (api APIv0) symbols(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	var js []byte
	err := logobj.Symbols().Save(func(data types.SymbolsData) error {
		var err error
		js, err = json.Marshal(&data)
		return err
	})
	if err != nil {
		api.serverError(w, err, "unknown error")
		return
	}
	api.write(w, js)
}
func (api APIv0) goModule(w http.ResponseWriter, r *http.Request) {
	logobj, pc, ok := api.getLogPC(w, r)
	if !ok {
		return
	}

	m, ok := logobj.Symbols().GoModule(pc)
	if !ok {
		http.Error(w, "not found module", http.StatusNotFound)
		return
	}
	api.writeObj(w, m)
}
func (api APIv0) goFunc(w http.ResponseWriter, r *http.Request) {
	logobj, pc, ok := api.getLogPC(w, r)
	if !ok {
		return
	}

	f, ok := logobj.Symbols().GoFunc(pc)
	if !ok {
		http.Error(w, "not found function", http.StatusNotFound)
		return
	}
	api.writeObj(w, f)
}
func (api APIv0) goLine(w http.ResponseWriter, r *http.Request) {
	logobj, pc, ok := api.getLogPC(w, r)
	if !ok {
		return
	}

	l, ok := logobj.Symbols().GoLine(pc)
	if !ok {
		http.Error(w, "not found line", http.StatusNotFound)
	}
	api.writeObj(w, l)
}
func (api APIv0) tracers(w http.ResponseWriter, r *http.Request) {
	// TODO: これを実装する前に、どのトレーサが接続しているのか管理出来るようにする
}
func (api APIv0) tracer(w http.ResponseWriter, r *http.Request) {
	// TODO: これを実装する前に、どのトレーサが接続しているのか管理出来るようにする
	switch r.Method {
	case http.MethodDelete:
	case http.MethodGet:
	case http.MethodPut:
	}
}

func (api APIv0) notImpl(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusInternalServerError)
}

func (api APIv0) getLog(w http.ResponseWriter, r *http.Request) (*storage.Log, bool) {
	strId := mux.Vars(r)["log-id"]
	id, err := storage.LogID{}.Unhex(strId)
	if err != nil {
		http.Error(w, "invalid log-id", http.StatusBadRequest)
		return nil, false
	}

	logobj, ok := api.Storage.Log(id)
	if !ok {
		http.Error(w, "log not found", http.StatusNotFound)
		return nil, false
	}
	return logobj, true
}

// getLogPC returns Log object and PC.
// If request is invalid, getLogPC writes the error message and returns false.
func (api APIv0) getLogPC(w http.ResponseWriter, r *http.Request) (logobj *storage.Log, pc uintptr, ok bool) {
	logobj, ok = api.getLog(w, r)
	if !ok {
		return
	}

	var err error
	pc, err = parseUintptr(mux.Vars(r)["pc"])
	if err != nil {
		http.Error(w, "invalid pc parameter because pc is not unsigned integer", http.StatusBadRequest)
		return
	}
	ok = true
	return
}

func (api *APIv0) worker(parent context.Context, logobj *storage.Log) *APIWorker {
	group, ctx := errgroup.WithContext(parent)
	return &APIWorker{
		Api:        api,
		Args:       &api.RouterArgs,
		Logger:     api.Logger,
		BufferSize: 1 << 10,
		Logobj:     logobj,
		group:      group,
		ctx:        ctx,
	}
}

func (w *APIWorker) wait() error {
	return w.group.Wait()
}
func (w *APIWorker) readFuncLog(minId, maxId int64) *FuncLogAPIWorker {
	ch := make(chan *types.FuncLog, w.BufferSize)
	newctx, cancel := context.WithCancel(w.ctx)
	fw := &FuncLogAPIWorker{
		api:        w,
		inCh:       ch,
		stopReader: cancel,
		readCtx:    newctx,
		sortCtx:    w.ctx,
		sendCtx:    w.ctx,
	}

	w.group.Go(func() error {
		defer close(ch)
		defer log.Print("readFuncLog: done")
		log.Println("readFuncLog: read from file")
		var err error
		w.Logobj.FuncLog(func(store *storage.FuncLogStore) {
			n := store.Records()
			if minId < 0 {
				minId = 0
			}
			if maxId < 0 || n < maxId {
				maxId = n
			}
			log.Printf("readFuncLog: start")
			log.Printf("readFuncLog: minId=%d maxId=%d", minId, maxId)
			for i := minId; i < maxId; i++ {
				fl := types.FuncLogPool.Get().(*types.FuncLog)
				err = store.GetNolock(types.FuncLogID(i), fl)
				if fl.Frames == nil {
					log.Panic("fl.Frames is nil", fl)
				}
				if err != nil {
					return
				}
				select {
				case ch <- fl:
				case <-fw.readCtx.Done():
					return
				}
			}
		})
		if err != nil {
			w.Logger.Println(errors.Wrap(err, "failed to read FuncLogFile"))
			return err
		}

		log.Println("readFuncLog: try to read from simulator")
		ss := w.Api.SimulatorStore.Get(w.Logobj.ID)
		if ss != nil {
			log.Println("readFuncLog: read from simulator")
			for _, fl := range ss.FuncLogs(true) {
				if fl.Frames == nil {
					log.Panic("fl.Frames is nil", fl)
				}
				if fl.ID < types.FuncLogID(maxId) {
					// 既に出力済みなのでスキップする
					continue
				}
				select {
				case ch <- fl:
				case <-fw.readCtx.Done():
					return nil
				}
			}
		}
		return nil
	})
	return fw
}

func (w *FuncLogAPIWorker) nextWorker(inCh chan *types.FuncLog) *FuncLogAPIWorker {
	worker := &FuncLogAPIWorker{}
	*worker = *w
	worker.inCh = inCh
	return worker
}

func (w *FuncLogAPIWorker) filterFuncLog(isFiltered func(evt *types.FuncLog) bool) *FuncLogAPIWorker {
	ch := make(chan *types.FuncLog, w.api.BufferSize)
	w.api.group.Go(func() error {
		log.Print("filterFuncLog: start")
		defer close(ch)
		defer log.Print("filterFuncLog: done")
		for {
			select {
			case evt, ok := <-w.inCh:
				if ok && !isFiltered(evt) {
					select {
					case ch <- evt:
					case <-w.readCtx.Done():
						return nil
					}
				} else {
					return nil
				}
			case <-w.readCtx.Done():
				return nil
			}
		}
	})
	return w.nextWorker(ch)
}

func (w *FuncLogAPIWorker) sortAndLimit(less func(f1, f2 *types.FuncLog) bool, limit int64) *FuncLogAPIWorker {
	ch := make(chan *types.FuncLog, w.api.BufferSize)

	if less == nil {
		// sortしない
		if limit <= 0 {
			// sortしない & limitしない。
			// つまり何もしない。
			return w
		} else {
			// 先頭からlimit個だけ取得して返す。
			w.api.group.Go(func() error {
				log.Print("limit: start")
				defer w.stopReader()
				defer close(ch)
				defer log.Println("limit: done")
				var i int64
				for {
					select {
					case evt, ok := <-w.inCh:
						if ok {
							ch <- evt
							i++
							if i >= limit {
								return nil
							}
						} else {
							return nil
						}
					case <-w.sortCtx.Done():
						return nil
					}
				}
				// unreachable
			})
			return w.nextWorker(ch)
		}
	}

	w.api.group.Go(func() error {
		start := time.Now()
		log.Print("sortAndLimit: start")
		defer w.stopReader()
		defer close(ch)
		defer log.Print("sortAndLimit: done exec-time=", time.Since(start))
		var items []*types.FuncLog

		// sort関数用の比較関数。
		sortComparator := func(i, j int) bool {
			return less(items[i], items[j])
		}
		// heap sortをするための比較関数。
		// heap内の値がより小さくなるようにするために、heapの先頭は最も大きな値が来るようにする。
		// そのため、比較関数のi, jを入れ替えている。
		heapComparator := func(j, i int) bool {
			return less(items[i], items[j])
		}

		if limit <= 0 {
			// read all items from input.
		ReadAllLoop:
			for {
				select {
				case evt, ok := <-w.inCh:
					if ok {
						items = append(items, evt)
					} else {
						break ReadAllLoop
					}
				case <-w.sortCtx.Done():
					return nil
				}
			}
			// sort all items.
			sort.Slice(items, sortComparator)
			// send all items to next worker.
			for i := range items {
				select {
				case ch <- items[i]:
				case <-w.sortCtx.Done():
					return nil
				}
			}
		} else {
			// read limited items from input.
			h := GenericHeap{
				LenFn:  func() int { return len(items) },
				LessFn: heapComparator,
				SwapFn: func(i, j int) { items[i], items[j] = items[j], items[i] },
				PushFn: func(x interface{}) { items = append(items, x.(*types.FuncLog)) },
				PopFn: func() interface{} {
					n := len(items)
					last := items[n-1]
					items = items[:n-1]
					return last
				},
			}

			// fill the items slice from inCh.
		FillItemsLoop:
			for int64(len(items)) < limit {
				select {
				case evt, ok := <-w.inCh:
					if ok {
						items = append(items, evt)
					} else {
						break FillItemsLoop
					}
				case <-w.sortCtx.Done():
					return nil
				}
			}
			heap.Init(&h)

		UpdateItemsLoop:
			for {
				select {
				case evt, ok := <-w.inCh:
					if ok {
						if less(evt, items[0]) {
							// replace a largest item with smaller item.
							items[0] = evt
							heap.Fix(&h, 0)
						}
					} else {
						break UpdateItemsLoop
					}
				case <-w.sortCtx.Done():
					return nil
				}
			}

			sort.Slice(items, sortComparator)
			for i := range items {
				select {
				case ch <- items[i]:
				case <-w.sortCtx.Done():
					return nil
				}
			}
		}
		return nil
	})
	return w.nextWorker(ch)
}

func (w *FuncLogAPIWorker) sendTo(enc Encoder) {
	w.api.group.Go(func() error {
		log.Print("sendTo: start")
		defer log.Print("sendTo: done")

		defer func() {
			// 途中で中断されたとき、チャンネルの中にいくつかのオブジェクトが滞留してしまう。
			// それを破棄せずに、poolに戻す。
			for evt := range w.inCh {
				types.FuncLogPool.Put(evt)
			}
		}()

		for {
			select {
			case evt, ok := <-w.inCh:
				if ok {
					if err := enc.Encode(evt); err != nil {
						w.api.Logger.Println(errors.Wrap(err, "failed to json.Encoder.Encode()"))
						return err
					}
					types.FuncLogPool.Put(evt)
				} else {
					return nil
				}
			case <-w.sendCtx.Done():
				return nil
			}
		}
		// unreachable
	})
}

func (h *GenericHeap) Len() int           { return h.LenFn() }
func (h *GenericHeap) Less(i, j int) bool { return h.LessFn(i, j) }
func (h *GenericHeap) Swap(i, j int)      { h.SwapFn(i, j) }
func (h *GenericHeap) Push(x interface{}) { h.PushFn(x) }
func (h *GenericHeap) Pop() interface{}   { return h.PopFn() }

func parseInt(value string, defaultValue int64) (int64, error) {
	if value == "" {
		return defaultValue, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return int64(intValue), nil
}

func parseTimestamp(value string, defaultValue types.Time) (types.Time, error) {
	if value == "" {
		return defaultValue, nil
	}
	var ts types.Time
	err := ts.UnmarshalJSON([]byte(value))
	if err != nil {
		return 0, err
	}
	return ts, nil
}

func parseSortKey(key string) (SortKey, error) {
	switch key {
	case string(NoSortKey):
		fallthrough
	case string(SortByID):
		fallthrough
	case string(SortByStartTime):
		fallthrough
	case string(SortByEndTime):
		return SortKey(key), nil
	default:
		return SortKey(""), fmt.Errorf("invalid sort key: %s", key)
	}
}

func parseOrder(order string, defaultOrder SortOrder) (SortOrder, error) {
	switch order {
	case string(NoSortOrder):
		return defaultOrder, nil
	case string(AscendingSortOrder):
		fallthrough
	case string(DescendingSortOrder):
		return SortOrder(order), nil
	default:
		return "", fmt.Errorf("invalid SortOrder: %s", order)
	}
}

func parseUintptr(s string) (uintptr, error) {
	ptr, err := strconv.ParseUint(s, 10, 64)
	return uintptr(ptr), err
}
