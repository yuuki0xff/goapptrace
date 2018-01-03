package restapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

type RouterArgs struct {
	Config  *config.Config
	Storage *storage.Storage
}

// Goapptrace REST API v0.xのハンドラを提供する
type APIv0 struct {
	RouterArgs
	Logger *log.Logger
}

type HttpRequestHandler func(w http.ResponseWriter, r *http.Request)
type APIRequestHandler func(w http.ResponseWriter, r *http.Request) (status int, data interface{}, err error)

// TODO: impl REST API server
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
	v01.HandleFunc("/log/{log-id}/status", api.log).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/status", api.log).Methods(http.MethodPut).
		Queries("version", "{version:[0-9]+}")
	v01.HandleFunc("/log/{log-id}/watch", api.notImpl).Methods(http.MethodGet).
		Queries(
			"version", "{version:[0-9]+}",
			"timeout", "{timeout:[0-9]+}",
		)

	v01.HandleFunc("/log/{log-id}/func-call/search", api.funcCallSearch).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/func-call/stream", api.notImpl).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/goroutines/search", api.goroutineSearch).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbol/func/{func-id}", api.funcSymbol).Methods(http.MethodGet)
	v01.HandleFunc("/log/{log-id}/symbol/func-status/{func-status-id}", api.funcStatusSymbol).Methods(http.MethodGet)

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

func (api APIv0) servers(w http.ResponseWriter, r *http.Request) {
	srvList := make([]*config.LogServerConfig, len(api.Config.Servers.LogServer))
	for _, srv := range api.Config.Servers.LogServer {
		srvList = append(srvList, srv)
	}

	js, err := json.Marshal(struct {
		Servers []*config.LogServerConfig `json:"servers"`
	}{
		srvList,
	})
	if err != nil {
		api.serverError(w, err, "failed to json.Marshal")
		return
	}
	api.write(w, js)
}
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
func (api APIv0) logs(w http.ResponseWriter, r *http.Request) {
	logs, err := api.Storage.Logs()
	if err != nil {
		api.serverError(w, err, "failed to load logs from storage")
		return
	}

	js, err := json.Marshal(struct {
		Logs []*storage.Log `json:"logs"`
	}{
		logs,
	})
	if err != nil {
		api.serverError(w, err, "failed to json.Marshal")
		return
	}
	api.write(w, js)
}
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
		api.notImpl(w, r)
	case http.MethodPut:
		api.notImpl(w, r)
	}
}
func (api APIv0) funcCallSearch(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	// suppress compile error
	_ = logobj
	// TODO: storage.LogにFuncLogの検索機能をつける
}
func (api APIv0) goroutineSearch(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	// suppress compile error
	_ = logobj
	// TODO: storage.LogにGoroutineの検索機能をつける
}
func (api APIv0) funcSymbol(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	strFid := mux.Vars(r)["func-id"]
	fid, err := strconv.Atoi(strFid)
	if err != nil {
		http.Error(w, "invalid func-id", http.StatusBadRequest)
		return
	}

	// suppress compile error
	_ = logobj
	_ = fid
	// TODO: SymbolsからFuncSymbolを取得するメソッドを追加する
	//logutil.FuncID(fid)
	//logobj.Symbols().
}
func (api APIv0) funcStatusSymbol(w http.ResponseWriter, r *http.Request) {
	logobj, ok := api.getLog(w, r)
	if !ok {
		return
	}

	strFsid := mux.Vars(r)["func-status-id"]
	fsid, err := strconv.Atoi(strFsid)
	if err != nil {
		http.Error(w, "invalid func-status-id", http.StatusBadRequest)
		return
	}

	// suppress compile error
	_ = logobj
	_ = fsid
	// TODO: SymbolsからFuncStatusを取得するメソッドを追加する
	//logutil.FuncID(fid)
	//logobj.Symbols().
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
