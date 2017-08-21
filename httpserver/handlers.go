package httpserver

import (
	"compress/gzip"
	"github.com/gorilla/mux"
	"github.com/yuuki0xff/goapptrace/log"
	"github.com/yuuki0xff/goapptrace/render"
	"io"
	"net/http"
	"os"
	"os/exec"
)

func getRouter() *mux.Router {
	router := mux.NewRouter()
	// TODO: Implements handlers

	if false {
		viewer := func(w io.Writer, args ...string) {
			f, _ := os.Open("/home/yuuki/work/docker-log/docker.log.21119.log.gz")
			g, _ := gzip.NewReader(f)
			arg := append([]string{"./viewer.py"}, args...)
			c := exec.Command("pypy3", arg...)
			c.Stdin = g
			c.Stdout = w
			c.Stderr = os.Stderr
			c.Start()
			c.Wait()
		}

		router.HandleFunc("/goroutines.svg", func(w http.ResponseWriter, r *http.Request) {
			viewer(w, "-l", "goroutines")
		})

		router.HandleFunc("/funccalls.svg", func(w http.ResponseWriter, r *http.Request) {
			viewer(w, "-l", "funccalls")
		})
	} else {
		f, _ := os.Open("/home/yuuki/work/docker-log/docker.log.21119.log.gz")
		g, _ := gzip.NewReader(f)
		l := log.Log{
			Name: "test",
		}
		if err := l.Load(g); err != nil {
			panic(err)
		}

		router.HandleFunc("/goroutines.svg", func(w http.ResponseWriter, r *http.Request) {
			rnd := render.SVGRender{
				StartTime: 0,
				EndTime:   100000,
				Layout:    render.Goroutine,
				Log:       &l,
				Height:    800,
				Colors: render.Colors{
					ColorRule: render.ColoringPerModule,
					NColors:   5,
				},
			}
			rnd.Render(w)
		})

		router.HandleFunc("/funccalls.svg", func(w http.ResponseWriter, r *http.Request) {
			rnd := render.SVGRender{
				StartTime: 0,
				EndTime:   100000,
				Layout:    render.FunctionCall,
				Log:       &l,
				Height:    800,
				Colors: render.Colors{
					ColorRule: render.ColoringPerModule,
					NColors:   5,
				},
			}
			rnd.Render(w)
		})
	}

	return router
}
