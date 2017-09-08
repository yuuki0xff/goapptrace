package httpserver

import (
	"compress/gzip"
	"net/http"
	"os"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/log"
	"github.com/yuuki0xff/goapptrace/tracer/render"
)

const BIT_SIZE = 64

func getRouter() *mux.Router {
	// prepare logs
	// TODO: 本来はHTTPリクエストを受け取った後に処理するべき
	f, _ := os.Open("/home/yuuki/work/docker-log/docker.log.21119.log.gz")
	g, _ := gzip.NewReader(f)
	l := log.RawLogLoader{
		Name: "test",
	}
	if err := l.LoadFromJsonLines(g); err != nil {
		panic(err)
	}

	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/log.svg", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// TOOD: error handling
		// TODO: scaleパラメータに対する処理を実装する
		width, _ := strconv.ParseInt(vars["width"], 10, BIT_SIZE)
		height, _ := strconv.ParseInt(vars["height"], 10, BIT_SIZE)
		scale, _ := strconv.ParseFloat(vars["scale"], BIT_SIZE)

		start, _ := strconv.ParseInt(vars["start"], 10, BIT_SIZE)
		end := start + int64(float64(width)*scale)

		layout := render.LayoutTypeNames[vars["layout"]]
		colorRule := render.ColorRuleNames[vars["color-rule"]]
		colors, _ := strconv.ParseInt(vars["colors"], 10, BIT_SIZE)

		rnd := render.SVGRender{
			StartTime: log.Time(start),
			EndTime:   log.Time(end),
			Layout:    layout,
			Log:       &l,
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
		http.FileServer(http.Dir(info.DEFAULT_HTTP_DOC_ROOT)))
	return router
}
