package httpserver

import (
	"net/http"

	"encoding/json"

	"log"

	"github.com/gorilla/mux"
	"github.com/yuuki0xff/goapptrace/info"
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

	router.PathPrefix("/").Methods("GET").Handler(
		http.FileServer(http.Dir(info.DocRootAbsPath)))
	return router
}
