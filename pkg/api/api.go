package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"goNews/pkg/db"
	"net/http"
	"strconv"
)

type API struct {
	r  *mux.Router
	db *db.DB
}

func New(db *db.DB) *API {
	api := API{}
	api.db = db
	api.r = mux.NewRouter()
	api.endpoints()
	return &api
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.HandleFunc("/news/{col}", api.ordersHandler).Methods(http.MethodGet)
	api.r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./src/webapp"))))
}

func (api *API) ordersHandler(w http.ResponseWriter, r *http.Request) {
	s := mux.Vars(r)["col"]
	col, err := strconv.Atoi(s)
	if err != nil || col != 0 {
		w.WriteHeader(http.StatusBadRequest)
	}
	orders := api.db.News(col)
	// Отправка данных клиенту в формате JSON.
	err = json.NewEncoder(w).Encode(orders)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
}
