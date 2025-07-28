package server

import (
	"encoding/json"
	"log"
	"net/http"

	"key-value-store/internal/config"
	"key-value-store/internal/raft"
	"key-value-store/internal/store"
)

// Dependencies for the HTTP server
type Server struct {
	kv       *store.KVStore
	raftNode *raft.RaftNode
}

// New Server instance
func New(kv *store.KVStore, raftNode *raft.RaftNode) *Server {
	return &Server{
		kv:       kv,
		raftNode: raftNode,
	}
}

// Stars the server
func (server *Server) Start(cfg *config.Config) {
	// Router for public client api
	publicMux := http.NewServeMux() // multiplexer
	publicMux.HandleFunc("/get", server.handleGet)
	publicMux.HandleFunc("/set", server.handleSet)
	publicMux.HandleFunc("/delete", server.handleDelete)

	// Internal Raft RPCs
	raftMux := http.NewServeMux()
	raftMux.HandleFunc("/requestVote", server.handleRequestVote)
	raftMux.HandleFunc("/appendEntries", server.handleAppendEntries)

	go func() {
		log.Printf("Public API server listening on %s", cfg.HTTPAddr)
		if err := http.ListenAndServe(cfg.HTTPAddr, publicMux); err != nil {
			log.Fatalf("Pubic server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("Internal API server listening on %s", cfg.RaftAddr)
		if err := http.ListenAndServe(cfg.RaftAddr, raftMux); err != nil {
			log.Fatalf("Raft server failed: %v", err)
		}
	}()
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
	var cmd raft.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	cmd.Op = "set"
	if !server.raftNode.SubmitCommand(cmd) {
		http.Error(w, "Not the leader. Try another node.", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Command successfully submitted to the cluster\n"))
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

	// Create a delete command
	cmd := raft.Command{
		Op:  "delete",
		Key: key,
	}

	// Submit it to the Raft log
	if !server.raftNode.SubmitCommand(cmd) {
		http.Error(w, "Not the leader. Please try another node.", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Delete command successfully submitted to the cluster.\n"))
}

func (server *Server) handleRequestVote(w http.ResponseWriter, r *http.Request) {
	var args raft.RequestVoteArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	reply := server.raftNode.HandleRequestVote(&args)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&reply); err != nil {
		http.Error(w, "Failed to encode reply", http.StatusInternalServerError)
	}
}

func (server *Server) handleAppendEntries(w http.ResponseWriter, r *http.Request) {
	var args raft.AppendEntriesArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reply := server.raftNode.HandleAppendEntries(&args)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reply); err != nil {
		http.Error(w, "Failed to encode reply", http.StatusInternalServerError)
	}
}
