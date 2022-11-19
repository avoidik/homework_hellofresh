package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// healthGetHandler handles GET /healthz
func (srv *WebServer) healthGetHandler(w http.ResponseWriter, r *http.Request) {

	// respond with error if no database connection
	if !srv.store.IsConnected() {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "notok")
		return
	}

	fmt.Fprint(w, "ok")
}

// defaultGetHandler handles GET /
func (srv *WebServer) defaultGetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "fresh-server - build "+tagRelease)
}

// configsGetAllHandler handles GET /configs
func (srv *WebServer) configsGetAllHandler(w http.ResponseWriter, r *http.Request) {
	cfgs, err := srv.store.GetConfigs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(cfgs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// configsPostHandler handles POST /configs
func (srv *WebServer) configsPostHandler(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	srv.store.InsertConfig(&cfg)
	fmt.Fprint(w, "new configuration item has successfully been added")
}

// configsGetOneHandler handles GET /configs/abc
func (srv *WebServer) configsGetOneHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if name == "" {
		http.Error(w, "configuration item was not found", http.StatusNotFound)
		return
	}

	cfg, err := srv.store.GetConfigByName(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "configuration item was not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// configsUpdateOneHandler handles PATCH /configs/abc
func (srv *WebServer) configsUpdateOneHandler(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := mux.Vars(r)["name"]
	if name == "" {
		http.Error(w, "configuration item was not found", http.StatusNotFound)
		return
	}

	srv.store.UpdateConfigByName(name, &cfg)

	fmt.Fprint(w, "new configuration item has successfully been updated")
}

// configsDeleteOneHandler handles DELETE /configs/abc
func (srv *WebServer) configsDeleteOneHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if name == "" {
		http.Error(w, "configuration item was not found", http.StatusNotFound)
		return
	}

	if err := srv.store.DeleteConfigByName(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "configuration item has successfully been erased")
}

// searchGetHandler handles GET /search
func (srv *WebServer) searchGetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Search GET!")
}

// initRoutes creates router for server
func (srv *WebServer) initRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/", srv.defaultGetHandler).Methods("GET")
	router.HandleFunc("/healthz", srv.healthGetHandler).Methods("GET")
	router.HandleFunc("/configs", srv.configsGetAllHandler).Methods("GET")
	router.HandleFunc("/configs", srv.configsPostHandler).Methods("POST")
	router.HandleFunc("/configs/{name}", srv.configsGetOneHandler).Methods("GET")
	router.HandleFunc("/configs/{name}", srv.configsUpdateOneHandler).Methods("PUT", "PATCH")
	router.HandleFunc("/configs/{name}", srv.configsDeleteOneHandler).Methods("DELETE")
	router.HandleFunc("/search", srv.searchGetHandler).Methods("GET")
	srv.Handler = router
}
