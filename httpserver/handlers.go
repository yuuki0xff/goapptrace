package httpserver

import (
	"net/http"

	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

const BIT_SIZE = 64

type ServerArgs struct {
	// TODO:
	Storage *storage.Storage
}

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
	api.HandleFunc("/log.svg", func(w http.ResponseWriter, r *http.Request) {
		//vars := mux.Vars(r)

		// TOOD: error handling
		// TODO: scaleパラメータに対する処理を実装する
		//width, _ := strconv.ParseInt(vars["width"], 10, BIT_SIZE)
		//height, _ := strconv.ParseInt(vars["height"], 10, BIT_SIZE)
		//scale, _ := strconv.ParseFloat(vars["scale"], BIT_SIZE)
		//
		//start, _ := strconv.ParseInt(vars["start"], 10, BIT_SIZE)
		//end := start + int64(float64(width)*scale)
		//
		//layout := render.LayoutTypeNames[vars["layout"]]
		//colorRule := render.ColorRuleNames[vars["color-rule"]]
		//colors, _ := strconv.ParseInt(vars["colors"], 10, BIT_SIZE)
		//
		//rnd := render.SVGRender{
		//	StartTime: log.Time(start),
		//	EndTime:   log.Time(end),
		//	Layout:    layout,
		//	Log:       &l,
		//	Height:    int(height),
		//	Colors: render.Colors{
		//		ColorRule: colorRule,
		//		NColors:   int(colors),
		//	},
		//}
		//w.Header().Add("Content-Type", "image/svg+xml")
		//rnd.Render(w)
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
		http.FileServer(http.Dir(info.DEFAULT_HTTP_DOC_ROOT)))
	return router
}
