/*
	The new file destination needed to be passed to every function that is because, it provides testability and flexibility
*/

package store

import (
	"bufio"
	"encoding/json"
	"os"
)

// A single log
type LogEntry struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

/*
marshaling refers to converting a Go data structure (like a struct) into a format like JSON, XML, Binary, Gob (Go's binary format)
re-marshaling...
*/
func (e *LogEntry) Write(file *os.File) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	// _ is the no of bytes written
	_, err = file.Write(append(data, '\n'))
	return err
}

func ReplayLog(file *os.File, store *KVStore) error {
	scanner := bufio.NewScanner(file)

	// scan checks next token to be recived
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return err
		}

		switch entry.Op {
		case "set":
			store.data[entry.Key] = entry.Value
		case "delete":
			delete(store.data, entry.Key)
		}
	}

	return scanner.Err()
}
