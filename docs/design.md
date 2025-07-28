## `DESIGN.md`

```markdown
# Project Design and Concepts

This document covers the core concepts and design decisions made during the development of the Go-Raft-KV store. It builds upon the foundational knowledge acquired during the project.

---

## 1. Core Concurrency and Networking

### Go's Concurrency Model (CSP)

This project heavily relies on Go's native concurrency features, which are based on the **Communicating Sequential Processes (CSP)** model. Instead of sharing memory and protecting it with locks (though mutexes are used for state protection within a node), the broader system design favors communication over explicit channels.

* **Goroutines:** Every concurrent task, from running an HTTP server to sending an RPC, is handled by a goroutine. This allows a node to manage thousands of concurrent operations efficiently without the overhead of traditional threads.
* **Channels:** While not used for the primary RPC mechanism (which is HTTP-based), the concept of channels informs the design. Goroutines are launched as independent processes that communicate over a shared medium (the network), which is a physical manifestation of the CSP philosophy.

### Network Communication (Duplex and RPC)

* **Full-Duplex Communication:** The underlying TCP/IP network used by our HTTP servers provides full-duplex communication, allowing nodes to send and receive data simultaneously.
* **Remote Procedure Calls (RPC):** The interaction between Raft nodes (`RequestVote`, `AppendEntries`) is implemented as an RPC system over HTTP. A node "calls" a function on a remote machine as if it were a local one, abstracting away the underlying network requests.

---

## 2. Raft Consensus Algorithm

Raft is a consensus algorithm designed to be understandable and efficient. It ensures that all nodes in a distributed system agree on the same sequence of operations, making the system fault-tolerant.

Our implementation includes the three core components of Raft:

### a. Leader Election

Nodes exist in one of three states: **Follower**, **Candidate**, or **Leader**.

1.  All nodes start as Followers.
2.  If a Follower doesn't receive a heartbeat from a leader within a randomized **election timeout**, it transitions to the Candidate state.
3.  As a Candidate, it increments the global **term** (a logical clock), votes for itself, and sends `RequestVote` RPCs to all other nodes.
4.  It becomes the Leader if it receives votes from a **majority** of the cluster.
5.  If another node establishes leadership first, it reverts to being a Follower.

This process guarantees that only one leader can be elected per term.

### b. Log Replication

Once a leader is elected, all client operations are funneled through it.

1.  A client sends a command (`set` or `delete`) to the leader.
2.  The leader appends the command to its internal **log** as a new entry, tagged with its current term.
3.  The leader sends the new log entries to all followers via the `AppendEntries` RPC. These RPCs also act as **heartbeats** to maintain the leader's authority.
4.  When an entry is replicated on a majority of nodes, the leader considers it **committed**.
5.  Once committed, each node **applies** the command from the log entry to its local state machine (the key-value store).

### c. Safety Protocols

To ensure correctness, several safety rules were implemented:

* **Voting Safety:** A node will only vote for a candidate if the candidate's log is at least as up-to-date as its own. This prevents a node with a stale log from becoming leader.
* **Log Consistency Check:** Followers only accept `AppendEntries` if the leader's log matches their own at the point where new entries are being added. If not, the leader finds the point of agreement and sends the correct logs to bring the follower up-to-date.

---

## 3. System Architecture

### Directory Structure

The project follows standard Go conventions for a clean, scalable structure:
* **`cmd/`**: Contains the main application entry points for the server (`dkv-server`) and the client (`dkv-cli`).
* **`internal/`**: Contains the core application logic, which is not meant to be imported by other projects.
    * **`config/`**: Logic for loading and parsing YAML configuration files.
    * **`store/`**: The underlying storage engine with its Write-Ahead Log.
    * **`server/`**: The HTTP server logic, responsible for routing public API and internal RPC traffic.
    * **`raft/`**: The Raft consensus module, containing all the core state and logic for elections and replication.
* **`configs/`**: Sample YAML configuration files for running a multi-node cluster.