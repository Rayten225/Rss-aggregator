package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"goNews/pkg/db"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type API struct {
	r  *mux.Router
	db *db.DB
}

func New(db *db.DB, errChan chan<- error) *API {
	if db == nil {
		errChan <- fmt.Errorf("database instance is nil")
		return nil
	}

	api := &API{db: db, r: mux.NewRouter()}
	api.endpoints(errChan)
	return api
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints(errCn chan<- error) {
	api.r.HandleFunc("/news/{col}", api.ordersHandler).Methods(http.MethodGet)

	webappPath := filepath.Join(".", "src", "webapp")
	if _, err := os.Stat(webappPath); os.IsNotExist(err) {
		errCn <- fmt.Errorf("static files directory not found: %s", webappPath)
		return
	}

	fs := http.FileServer(http.Dir(webappPath))
	api.r.PathPrefix("/").Handler(http.StripPrefix("/", fs))
}

func (api *API) ordersHandler(w http.ResponseWriter, r *http.Request) {
	errcol := make([]string, 0)
	s := mux.Vars(r)["col"]
	if s == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errcol)
	}
	col, err := strconv.Atoi(s)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errcol)
		return
	}
	if col < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errcol)
		return
	}

	news, err := api.db.News(r.Context(), col)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch news: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(news); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
