package store

import (
	"errors"
	"log"
	"os"
	"sync"
)

const logFileName = "kv.log"

type KVStore struct {
	mutex   sync.RWMutex // Read Write Mutex
	data    map[string]string
	logFile *os.File
}

func CreateKVStore() *KVStore {
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	newStore := &KVStore{
		data:    make(map[string]string),
		logFile: file,
	}

	log.Println("Replaying log...")
	if err := ReplayLog(file, newStore); err != nil {
		log.Fatalf("Failed to replay log: %v", err)
	}
	log.Println("Log replay finished.")

	return newStore
}

// SET:
func (store *KVStore) Set(key, value string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Write to log file
	entry := LogEntry{Op: "set", Key: key, Value: value}
	if err := entry.Write(store.logFile); err != nil {
		log.Printf("Failed to write to log: %v", err)
		return
	}
	store.data[key] = value
}

// GET:
func (store *KVStore) Get(key string) (string, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	value, ok := store.data[key]
	if !ok {
		return "", errors.New("key not found")
	}

	return value, nil
}

// DELETE:
func (store *KVStore) Delete(key string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	_, ok := store.data[key]
	if !ok {
		log.Print("Key is not in the store")
		return
	}

	// Write to log file
	entry := LogEntry{Op: "delete", Key: key}
	if err := entry.Write(store.logFile); err != nil {
		log.Printf("Failed to write to log: %v", err)
		return
	}

	delete(store.data, key)
}
