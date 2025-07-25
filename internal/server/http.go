package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"key-value-store/internal/config"
	"key-value-store/internal/store"
)

// Dependencies for the HTTP server
type Server struct {
	kv  *store.KVStore
	cfg *config.Config
}

// New Server instance
func New(kv *store.KVStore, cfg *config.Config) *Server {
	return &Server{
		kv:  kv,
		cfg: cfg,
	}
}

// Stars the server
func (server *Server) Start(addr string) error {
	// Public API
	http.HandleFunc("/get", server.handleGet)
	http.HandleFunc("/set", server.handleSet)
	http.HandleFunc("/delete", server.handleDelete)

	// Internal Api
	http.HandleFunc("/replicate", server.handleReplicate)

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
	go server.replicate(store.LogEntry{Op: "set", Key: data.Key, Value: data.Value})
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
	go server.replicate(store.LogEntry{Op: "delete", Key: key})
	w.WriteHeader(http.StatusOK)
}

func (server *Server) handleReplicate(w http.ResponseWriter, r *http.Request) {
	var entry store.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid Json Body", http.StatusBadRequest)
		return
	}

	log.Printf("Replicating operation %s for %s", entry.Op, entry.Key)

	switch entry.Op {
	case "set":
		server.kv.Set(entry.Key, entry.Value)
	case "delete":
		server.kv.Delete(entry.Key)
	default:
		http.Error(w, "Invalid Operation", http.StatusBadRequest)
	}
}

func (server *Server) replicate(entry store.LogEntry) {
	var wg sync.WaitGroup
	for _, peerAddr := range server.cfg.Peers {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			url := fmt.Sprintf("http://%s/replicate", addr)
			jsonData, err := json.Marshal(entry)
			if err != nil {
				log.Printf("Failed to marshal replication entry: %v", err)
				return
			}

			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("Failed to replicate peer %s: %v", peerAddr, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("Peer %s returned non-Ok status: %s", peerAddr, resp.Status)
			}
		}(peerAddr)
	}
	wg.Wait()
}
