# A Fault-Tolerant Distributed Key-Value Store

![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

Go-Raft-KV is a distributed, fault-tolerant key-value store built from scratch in Go. It implements the Raft consensus algorithm to ensure data consistency and high availability across a cluster of nodes. The project demonstrates a deep, practical understanding of distributed systems principles, concurrency, and network programming.

---

## Key Features

* **Raft Consensus Implementation:** The core of the project is a from-scratch implementation of the Raft algorithm, including:
    * Leader Election with randomized timeouts.
    * Log Replication for command synchronization.
    * Safety protocols to ensure data consistency.
* **Fault Tolerance:** The cluster can automatically recover from leader failure by electing a new leader, ensuring the service remains available.
* **Distributed Key-Value Store API:** A simple HTTP API (`GET`, `SET`, `DELETE`) for interacting with the store.
* **Separated Network Traffic:** Uses separate network ports for the public client API and internal Raft RPCs, mimicking production-grade architecture for better performance and security.
* **Smart CLI Client:** A command-line client that can automatically discover the cluster leader and forward requests, simplifying user interaction.

---

## Project Architecture

The system is designed with a clean separation of concerns, divided into three main internal modules:

1.  **`store`:** The underlying key-value storage engine. It uses an in-memory map protected by a mutex and a Write-Ahead Log (WAL) for on-disk persistence.
2.  **`raft`:** The consensus module. This is the heart of the system, managing node states (Follower, Candidate, Leader), terms, log replication, and leader election.
3.  **`server`:** The networking layer. It runs two parallel HTTP servers: one for the public API and another for the internal Raft RPCs (`RequestVote`, `AppendEntries`).

---

## Technology Stack

* **Language:** Go
* **Concurrency:** Go routines, Channels, and `sync.Mutex` for managing concurrent operations.
* **Networking:** Go's standard `net/http` library for both the public API and internal RPCs.
* **Configuration:** YAML for clean, readable node configuration.

---

## Getting Started

### Prerequisites

* Go 1.18 or later.

### Configuration

The cluster configuration is defined in the `configs/` directory. The project includes sample files (`node1.yaml`, `node2.yaml`, `node3.yaml`) for a three-node cluster.

### Running the Cluster

Open three separate terminals to run each node of the cluster.

**Terminal 1:**
```bash
go run cmd/dkv-server/main.go --config configs/node1.yaml
```

**Terminal 2:**

```bash
go run cmd/dkv-server/main.go --config configs/node2.yaml
```

**Terminal 3:**

```bash
go run cmd/dkv-server/main.go --config configs/node3.yaml
```

After a few moments, the nodes will elect a leader and the cluster will be ready.

### Using the CLI Client

The smart client can interact with the cluster without needing to know which node is the leader.

**Set a value:**

```bash
go run cmd/dkv-cli/main.go --op=set --key=hello --value=distributed-world
```

**Get a value (using `curl`):**

```bash
curl http://localhost:8081/get?key=hello
```

**Delete a value:**

```bash
go run cmd/dkv-cli/main.go --op=delete --key=hello
```

-----

## Future Enhancements

  * **Persist Raft State:** Save the `currentTerm`, `votedFor`, and the Raft `log` itself to disk to allow for full recovery after a restart.
  * **Log Compaction (Snapshotting):** Implement snapshotting to prevent the Raft log from growing indefinitely.
  * **Dynamic Cluster Membership:** Add support for adding or removing nodes from a running cluster.
