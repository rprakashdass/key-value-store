package server

import (
	"encoding/json"
	"log"
	"net/http"

	"key-value-store/internal/store"
)

// Dependencies for the HTTP server
type Server struct {
	kv *store.KVStore
}

// New Server instance
func New(kv *store.KVStore) *Server {
	return &Server{kv: kv}
}

// Stars the server
func (server *Server) Start(addr string) error {
	http.HandleFunc("/get", server.handleGet)
	http.HandleFunc("/set", server.handleSet)
	http.HandleFunc("/delete", server.handleDelete)

	log.Printf("Server listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleGet handles retrieving a key
// GET /get?key=mykey
func (server *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key query parameter", http.StatusBadRequest)
		return
	}

	value, err := server.kv.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Write([]byte(value))
}

// handleSet handles setting a key-value pair
// POST /set with JSON body {"key": "mykey", "value": "myvalue"}
func (server *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var data struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	server.kv.Set(data.Key, data.Value)
	w.WriteHeader(http.StatusOK)
}

// handleDelete handles deleting a key
// DELETE /delete?key=mykey
func (server *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key query parameter", http.StatusBadRequest)
		return
	}

	server.kv.Delete(key)
	w.WriteHeader(http.StatusOK)
}