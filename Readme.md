# A Fault-Tolerant Distributed Key-Value Store

![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

This project is a distributed, fault-tolerant key-value store built from the ground up in Go. It leverages a consensus algorithm to ensure data consistency and high availability across a cluster of nodes, demonstrating a practical application of distributed systems principles, advanced concurrency, and network programming.

---
## Table of Contents
- [A Fault-Tolerant Distributed Key-Value Store](#a-fault-tolerant-distributed-key-value-store)
  - [Table of Contents](#table-of-contents)
  - [About The Project](#about-the-project)
  - [Key Features](#key-features)
  - [System Architecture](#system-architecture)
  - [Core Implementation Details](#core-implementation-details)
  - [Technology Stack](#technology-stack)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Configuration](#configuration)
    - [Running the Cluster](#running-the-cluster)
    - [Using the CLI Client](#using-the-cli-client)
  - [Future Enhancements](#future-enhancements)
  - [🤝 Let's Connect!](#-lets-connect)
  - [License](#license)

---
## About The Project

Modern applications demand high availability and data consistency. A service running on a single server presents a single point of failure, while managing multiple data copies without a rigorous protocol can lead to data integrity issues.

This key-value store solves these challenges. It operates as a coordinated cluster where nodes work together to form a single, logical, and reliable data service. By using a consensus protocol, it guarantees that even if some nodes fail, the system remains online and all data remains correct and consistent across the cluster.

---
## Key Features

* **Consensus-Based Replication:** At its core is a from-scratch implementation of the Raft consensus algorithm, ensuring all changes to the data are agreed upon by a majority of the cluster before being committed.
* **High Availability & Fault Tolerance:** The cluster can withstand multiple node failures (up to 2 in a 5-node cluster). It automatically detects when a leader node is down, holds a new election, and promotes a new leader without manual intervention.
* **Strong Consistency:** All write operations are linearized through the elected leader, guaranteeing that all clients have a consistent view of the data.
* **Production-Style Network Architecture:** The system uses separate network ports for the public client-facing API and the private, internal cluster communication, a design that enhances security and performance.
* **Intelligent CLI Client:** A command-line client is included, which features automatic leader discovery. If a request is sent to a non-leader node, the client intelligently redirects the request to the true leader.

---
## System Architecture

The project is designed with a clean separation of concerns into distinct, modular components:

1.  **Storage Layer (`store`):** The underlying key-value storage engine. It uses a thread-safe, in-memory map and a Write-Ahead Log (WAL) for on-disk persistence, ensuring data survives server restarts.
2.  **Consensus Layer (`raft`):** The brain of the system. This module manages node states, handles the entire leader election process, and orchestrates the replication of commands across the cluster.
3.  **Networking Layer (`server`):** This layer runs two parallel HTTP servers on each node: one for the public API (`GET`, `SET`, `DELETE`) and another for the private, internal consensus RPCs (`RequestVote`, `AppendEntries`).

---
## Core Implementation Details

The consensus module operates based on several key mechanics:

* **Leader Election and Voting:** All nodes start in the `Follower` state. If a follower does not receive a heartbeat from a leader within a randomized election timeout, it transitions to the `Candidate` state. As a candidate, it increments the current term, votes for itself, and sends `RequestVote` RPCs to all other nodes. A node will grant its vote only if the candidate's log is at least as up-to-date as its own. If the candidate receives votes from a majority of the cluster, it becomes the `Leader`.

* **Log Replication and Heartbeats:** The elected leader is responsible for managing all state changes. It periodically sends `AppendEntries` RPCs to all followers to serve as heartbeats, preventing them from starting new elections. When a client issues a write command (`set`, `delete`), the leader appends the command to its internal log and sends it to followers within the next `AppendEntries` RPC. Each RPC includes the preceding log index and term to ensure followers' logs are consistent with the leader's.

* **Commit and Apply Mechanism:** An entry in the leader's log is considered **committed** once it has been successfully replicated to a majority of the nodes. The leader tracks this commit index and informs followers of its progress via the `AppendEntries` RPCs. On each node, a dedicated background goroutine (the **applier**) constantly checks for newly committed entries and applies them in order to its local key-value store, ensuring the state machine reflects the agreed-upon log.

---
## Technology Stack

* **Language:** Go
* **Concurrency:** Goroutines & Mutexes
* **Networking:** Go's standard `net/http` library
* **Configuration:** YAML

---
## Getting Started

### Prerequisites
* Go 1.18 or later installed.
* A terminal that supports running multiple concurrent sessions (e.g., using tabs or split panes).

### Configuration
The `configs/` directory contains sample configurations for a 5-node cluster. Please ensure the files `node1.yaml` through `node5.yaml` are present and correctly configured before proceeding.

### Running the Cluster
Open five separate terminals to run each node of the cluster.

* **Terminal 1:** `go run cmd/dkv-server/main.go --config configs/node1.yaml`
* **Terminal 2:** `go run cmd/dkv-server/main.go --config configs/node2.yaml`
* **Terminal 3:** `go run cmd/dkv-server/main.go --config configs/node3.yaml`
* **Terminal 4:** `go run cmd/dkv-server/main.go --config configs/node4.yaml`
* **Terminal 5:** `go run cmd/dkv-server/main.go --config configs/node5.yaml`

After a few moments, the nodes will elect a leader and the cluster will be ready.

### Using the CLI Client
Once the cluster is running, open a sixth terminal to interact with it.

1.  **Build the Client:**
    ```bash
    go build -o dkv-cli cmd/dkv-cli/main.go
    ```
2.  **Use the Client:**
    ```bash
    # Set a value
    ./dkv-cli set mykey "hello-distributed-world"

    # Get a value
    ./dkv-cli get mykey
    
    # Delete a value
    ./dkv-cli delete mykey
    ```

---
## Future Enhancements

* **Persist Raft State:** Save the consensus-critical state (`currentTerm`, `votedFor`, `log`) to disk to enable full recovery from a total cluster shutdown.
* **Log Compaction (Snapshotting):** Implement snapshotting to prevent the Raft log from growing indefinitely in long-running deployments.
* **Dynamic Cluster Membership:** Add support for adding or removing nodes from a running cluster without downtime.

---
## 🤝 Let's Connect\!

I'd love to discuss this project further or any opportunities you might have.
  * **Website:** https://www.rprakashdass.in/
  * **GitHub:** https://github.com/rprakashdass
  * **LinkedIn:** https://linkedin.com/in/rprakashdass
  * **Email:** rprakashdass@gmail.com

---
## License

This project is licensed under the MIT License.