package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	nodes []string
	http  *http.Client
}

func NewClient(nodes string) *Client {
	return &Client{
		nodes: strings.Split(nodes, ","),
		http:  &http.Client{},
	}
}

// request sends an HTTP request to the leader
func (c *Client) request(method, path string, payload io.Reader) ([]byte, error) {
	/*
		hints: the non leader nodes provides this as leader nodes address
	*/
	// Start with the first known node as the target
	targetNode := c.nodes[0]

	// trying a few times, following leader hints by non leader nodes
	for attempts := 0; attempts < len(c.nodes)+2; attempts++ {
		url := fmt.Sprintf("http://%s%s", targetNode, path)
		log.Printf("Attempting %s request to %s", method, url)

		req, err := http.NewRequest(method, url, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			// This node is likely down, Try the next one in the list
			log.Printf("Node %s is unreachable, trying next. Error: %v", targetNode, err)
			targetNode = c.nodes[(attempts+1)%len(c.nodes)]
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		// Success! The node processed our request
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		// The node is not the leader, It will give us a hint
		if resp.StatusCode == http.StatusServiceUnavailable {
			var errResp map[string]string
			if json.Unmarshal(body, &errResp) == nil {
				if leaderAddr, ok := errResp["leader_addr"]; ok && leaderAddr != "" {
					log.Printf("Not the leader. New leader hint: %s", leaderAddr)
					targetNode = leaderAddr
					continue
				}
			}
		}

		// Handle other HTTP errors
		return nil, fmt.Errorf("request failed: status %s, body: %s", resp.Status, string(body))
	}

	return nil, errors.New("failed to complete request after multiple attempts")
}

func main() {
	nodesFlag := flag.String("nodes", "localhost:8081,localhost:8082,localhost:8083,localhost:8084,localhost:8085", "Comma-separated list of cluster API nodes")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Usage: dkv-cli --nodes=<...> <op> <key> [value]")
		os.Exit(1)
	}

	client := NewClient(*nodesFlag)
	op := flag.Arg(0)
	key := flag.Arg(1)

	switch op {
	case "get":
		body, err := client.request("GET", "/get?key="+key, nil)
		if err != nil {
			log.Fatalf("GET operation failed: %v", err)
		}
		fmt.Println(string(body))

	case "set":
		if flag.NArg() < 3 {
			log.Fatal("`set` operation requires a value.")
		}
		value := flag.Arg(2)
		payload, _ := json.Marshal(map[string]string{"key": key, "value": value})
		_, err := client.request("POST", "/set", bytes.NewBuffer(payload))
		if err != nil {
			log.Fatalf("SET operation failed: %v", err)
		}
		fmt.Println("OK")

	case "delete":
		_, err := client.request("DELETE", "/delete?key="+key, nil)
		if err != nil {
			log.Fatalf("DELETE operation failed: %v", err)
		}
		fmt.Println("OK")

	default:
		log.Fatalf("Invalid operation: %s. Must be 'get', 'set', or 'delete'.", op)
	}
}
