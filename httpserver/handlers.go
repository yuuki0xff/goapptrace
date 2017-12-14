package httpserver

import (
	"net/http"

	"encoding/json"

	"strconv"

	"time"

	"log"

	"github.com/gorilla/mux"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/render"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

const BIT_SIZE = 64

type ServerArgs struct {
	// TODO:
	Storage *storage.Storage
}

// TODO: define schema for API response
func getRouter(args *ServerArgs) *mux.Router {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		logs, err := args.Storage.Logs()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		logsRes := []interface{}{}
		for _, log := range logs {
			logsRes = append(logsRes, struct {
				ID string
			}{
				ID: log.ID.Hex(),
			})
		}
		res := struct {
			Logs interface{}
		}{
			Logs: logsRes,
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			panic(err)
		}
	})
	api.HandleFunc("/log/{id:[0-9a-f]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		strid := vars["id"]
		id, err := storage.LogID{}.Unhex(strid)
		if err != nil {
			log.Printf("INFO: invalid id: %s\n", strid)
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		logobj, ok := args.Storage.Log(id)
		if !ok {
			http.Error(w, "not found Log", http.StatusNotFound)
			return
		}

		res := struct {
			ID       string
			Metadata *storage.LogMetadata
		}{
			ID:       logobj.ID.Hex(),
			Metadata: logobj.Metadata,
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			panic(err)
		}
	})
	api.HandleFunc("/log/{id:[0-9a-f]+}.svg", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		strid := vars["id"]
		id, err := storage.LogID{}.Unhex(strid)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		logobj, ok := args.Storage.Log(id)
		if !ok {
			http.Error(w, "not found Log", http.StatusNotFound)
			return
		}
		reader, err := logobj.Reader()
		if err != nil {
			http.Error(w, "failed to initialization", http.StatusInternalServerError)
			return
		}
		defer reader.Close() // nolint: errcheck

		rawlog := &logutil.RawLogLoader{
			Name: strid,
		}
		rawlog.Init()
		log.Printf("DEBUG: logobj symbols: %+v\n", reader.Symbols())
		rawlog.SymbolsEditor.AddSymbols(reader.Symbols())
		log.Printf("DEBUG: rawlog symbols: %+v\n", &rawlog.Symbols)

		// TODO: error handling
		// TODO: scaleパラメータに対する処理を実装する
		width, _ := strconv.ParseInt(vars["width"], 10, BIT_SIZE)
		height, _ := strconv.ParseInt(vars["height"], 10, BIT_SIZE)
		scale, _ := strconv.ParseFloat(vars["scale"], BIT_SIZE)

		start, _ := strconv.ParseInt(vars["start"], 10, BIT_SIZE)
		end := start + int64(float64(width)*scale)

		layout := render.LayoutTypeNames[vars["layout"]]
		colorRule := render.ColorRuleNames[vars["color-rule"]]
		colors, _ := strconv.ParseInt(vars["colors"], 10, BIT_SIZE)

		// TODO:
		logChan := make(chan logutil.RawFuncLog, 10000)
		go func() {
			if err := reader.Search(time.Unix(start, 0), time.Unix(end, 0), func(evt logutil.RawFuncLog) error {
				logChan <- evt
				return nil
			}); err != nil {
				panic(err)
			}
			close(logChan)
		}()
		if err := rawlog.LoadFromIterator(func() (raw logutil.RawFuncLog, ok bool) {
			raw, ok = <-logChan
			if !ok {
				return
			}
			return
		}); err != nil {
			panic(err)
		}
		rnd := render.SVGRender{
			StartTime: logutil.Time(start),
			EndTime:   logutil.Time(end),
			Layout:    layout,
			Log:       rawlog,
			Height:    int(height),
			Colors: render.Colors{
				ColorRule: colorRule,
				NColors:   int(colors),
			},
		}
		w.Header().Add("Content-Type", "image/svg+xml")
		rnd.Render(w)
	}).Queries(
		"width", "{width:[0-9]+}",
		"height", "{height:[0-9]+}",

		"layout", "{layout:[a-z]+}",
		"color-rule", "{color-rule:[a-z]+}",
		"colors", "{colors:[0-9]+}",
		"start", "{start:[0-9]+}",
		"scale", "{scale:[0-9.]+}",
	)

	router.PathPrefix("/").Methods("GET").Handler(
		http.FileServer(http.Dir(info.DocRootAbsPath)))
	return router
}
