package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"key-value-store/internal/config"
	"key-value-store/internal/store"
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

type RaftState int

const (
	Follower RaftState = iota
	Candidate
	Leader
)

// AppendEntries RPC

type AppendEntriesArgs struct {
	Term              int
	LeaderID          string
	PrevLogIndex      int
	PrevLogTerm       int
	Entries           []LogEntry
	LeaderCommitIndex int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

// Commands

type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LogEntry struct {
	Term    int
	Command Command
}

// Raft
type RaftNode struct {
	mutex      sync.Mutex
	cfg        *config.Config
	localStore *store.KVStore

	// Persistent state (non-volatile) should be in disks
	currentTerm int
	votedFor    string
	log         []LogEntry

	// Volatile state (in-memory only)
	state         RaftState
	electionTime  time.Time
	votesReceived int
	commitIndex   int // index of last commited log
	lastApplied   int // index of last applied log

	// Leader specifi (volatile)
	matchIndex map[string]int // index of highest log entry known to be replicated with the leader
	nextIndex  map[string]int // index of next log entry that leader should send to followers
}

// Requests
type RequestVoteArgs struct {
	Term         int
	CandidateID  string
	Entries      []LogEntry
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func NewRaftNode(cfg *config.Config, store *store.KVStore) *RaftNode {
	raftNode := &RaftNode{
		cfg:        cfg,
		localStore: store,
		state:      Follower,
		log:        make([]LogEntry, 1),
	}
	raftNode.resetElectionTimer()
	go raftNode.run()
	go raftNode.applier()
	return raftNode
}

func (raftNode *RaftNode) run() {
	for {

		raftNode.mutex.Lock()
		state := raftNode.state
		timer := time.Now().After(raftNode.electionTime)
		currentTerm := raftNode.currentTerm

		switch state {
		case Follower:
			if timer {
				raftNode.mutex.Unlock()
				log.Printf("[%s] Election timeout reached, becoming a candidate for term %d", raftNode.cfg.NodeID, currentTerm)
				raftNode.becomeCandidate()
				continue
			}

		case Candidate:
			// Start an election
			if timer {
				raftNode.mutex.Unlock()
				log.Printf("[%s] Election timed out, starting new election.", raftNode.cfg.NodeID)
				raftNode.becomeCandidate()
				continue
			}

		case Leader:
			// Send heartbeats
			raftNode.mutex.Unlock()
			raftNode.sendAppendEntries()
			time.Sleep(50 * time.Millisecond)
			continue
		}

		raftNode.mutex.Unlock()           // if no stages changed
		time.Sleep(10 * time.Millisecond) // Short sleep to prevent busy-looping
	}
}

func (raftNode *RaftNode) applier() {
	for {

		raftNode.mutex.Lock()
		lastCommited := raftNode.commitIndex
		lastApplied := raftNode.lastApplied
		raftNode.mutex.Unlock()

		for i := lastApplied + 1; i <= lastCommited; i++ {
			entry := raftNode.log[i]
			log.Printf("[%s] Applying command to store: %+v", raftNode.cfg.NodeID, entry.Command)

			switch entry.Command.Op {
			case "set":
				raftNode.localStore.Set(entry.Command.Key, entry.Command.Value)
			case "delete":
				raftNode.localStore.Delete(entry.Command.Key)
			}

			raftNode.mutex.Lock()
			raftNode.lastApplied = i
			raftNode.mutex.Unlock()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (raftNode *RaftNode) HandleRequestVote(args *RequestVoteArgs) *RequestVoteReply {
	raftNode.mutex.Lock()
	defer raftNode.mutex.Unlock()

	reply := &RequestVoteReply{
		Term:        raftNode.currentTerm,
		VoteGranted: false,
	}

	if args.Term < raftNode.currentTerm {
		return reply
	}

	if args.Term > raftNode.currentTerm {
		raftNode.becomeFollower(args.Term)
	}

	if raftNode.votedFor == "" || raftNode.votedFor == args.CandidateID {
		// Raft log safety check (up to date)
		lastLogIndex := len(raftNode.log) - 1
		lastLogTerm := raftNode.log[lastLogIndex].Term

		if args.LastLogIndex > lastLogIndex || args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogTerm {
			raftNode.votedFor = args.CandidateID
			reply.VoteGranted = true
			raftNode.resetElectionTimer()
			log.Printf("[%s] Voted for %s in term %d.", raftNode.cfg.NodeID, args.CandidateID, raftNode.currentTerm)
		} else {
			log.Printf("[%s] Rejected vote for %s because its log is not up to date", raftNode.cfg.NodeID, args.CandidateID)
		}
	}
	return reply
}

func (raftNode *RaftNode) becomeCandidate() {
	raftNode.mutex.Lock()         // lock
	defer raftNode.mutex.Unlock() // unlock
	raftNode.state = Candidate
	raftNode.currentTerm++
	raftNode.votedFor = raftNode.cfg.NodeID
	raftNode.votesReceived = 1
	raftNode.resetElectionTimer()
	term := raftNode.currentTerm
	go raftNode.startElection(term)
}

func (raftNode *RaftNode) becomeFollower(term int) {
	// lock is aldready held by startstartElection()
	raftNode.state = Follower
	raftNode.currentTerm = term
	raftNode.votedFor = ""
	raftNode.resetElectionTimer()
}

func (raftNode *RaftNode) becomeLeader() {
	raftNode.state = Leader
	raftNode.matchIndex = make(map[string]int)
	raftNode.nextIndex = make(map[string]int)
	logLen := len(raftNode.log)
	for _, peer := range raftNode.cfg.Peers {
		raftNode.matchIndex[peer] = 0
		raftNode.nextIndex[peer] = logLen
	}
	log.Printf("[%s] Became leader for term %d!", raftNode.cfg.NodeID, raftNode.currentTerm)
}

func (raftNode *RaftNode) resetElectionTimer() {
	/* Raft uses randomized timeouts to prevent split votes.
	   A typical range is 150-300ms.
	   Time Duration:
			min - 0 + 150 = 150
	   		max - 150 + 149 = 299
	*/
	timeout := time.Duration(150+rand.IntN(150)) * time.Millisecond // final result as millisecond
	raftNode.electionTime = time.Now().Add(timeout)
	log.Printf("New election timer set for %v", timeout)
}

func (raftNode *RaftNode) startElection(term int) {
	log.Printf("[%s] Starting election for term %d", raftNode.cfg.NodeID, raftNode.currentTerm)

	raftNode.mutex.Lock()
	lastLogIndex := len(raftNode.log) - 1
	lastLogTerm := raftNode.log[lastLogIndex].Term
	raftNode.mutex.Unlock()

	args := RequestVoteArgs{
		Term:         term,
		CandidateID:  raftNode.cfg.NodeID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	for _, peerAddr := range raftNode.cfg.Peers {
		go func(addr string) {
			reply := raftNode.sendRequestVote(addr, &args)
			if reply == nil {
				// sendRequestVote aldready logs the error
				return
			}

			raftNode.mutex.Lock()
			defer raftNode.mutex.Unlock()

			if reply.Term > raftNode.currentTerm {
				raftNode.becomeFollower(reply.Term)
				return
			}

			// Check if we are still a candidate for the same term
			if raftNode.state != Candidate || raftNode.currentTerm != term {
				return // we lost the election
			}

			if reply.VoteGranted {
				raftNode.votesReceived++ // self vote

				if raftNode.votesReceived > (len(raftNode.cfg.Peers)+1)/2 {
					raftNode.becomeLeader()
				}
			}

		}(peerAddr)
	}
}

func (raftNode *RaftNode) sendRequestVote(addr string, args *RequestVoteArgs) *RequestVoteReply {
	url := fmt.Sprintf("http://%s/requestVote", addr)
	jsonArgs, _ := json.Marshal(args)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonArgs))
	if err != nil {
		log.Printf("[%s] Failed to send RequestVote to %s: %v", raftNode.cfg.NodeID, addr, err)
		return nil
	}
	defer resp.Body.Close()

	var reply RequestVoteReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		log.Printf("[%s] Failed to decode RequestVote reply from %s: %v", raftNode.cfg.NodeID, addr, err)
		return nil
	}

	return &reply
}

func (raftNode *RaftNode) HandleAppendEntries(args *AppendEntriesArgs) *AppendEntriesReply {
	raftNode.mutex.Lock()
	defer raftNode.mutex.Unlock()

	reply := &AppendEntriesReply{
		Term:    raftNode.currentTerm,
		Success: false,
	}

	if args.Term < raftNode.currentTerm {
		return reply
	}

	if args.Term >= raftNode.currentTerm {
		log.Printf("[%s] Received heartbeat from new leader %s in same or higher term. Becoming follower.", raftNode.cfg.NodeID, args.LeaderID)
		raftNode.becomeFollower(args.Term)
	}

	// raft consistency check
	if args.PrevLogIndex < 0 ||
		args.PrevLogIndex >= len(raftNode.log) ||
		args.PrevLogTerm != raftNode.log[args.PrevLogIndex].Term {
		log.Printf("[%s] AppendEntries consistency failed", raftNode.cfg.NodeID)
		return reply
	}

	// append Entries
	if len(args.Entries) > 0 {
		conflictIndex := -1
		for i := 0; i < len(args.Entries); i++ {
			logIndex := args.PrevLogIndex + i + 1
			if logIndex >= len(raftNode.log) || raftNode.log[logIndex].Term != args.Entries[i].Term {
				conflictIndex = logIndex
				break
			}
		}
		if conflictIndex != -1 {
			log.Printf("[%s] Truncating log from index %d", raftNode.cfg.NodeID, conflictIndex)
			raftNode.log = raftNode.log[:conflictIndex]
			raftNode.log = append(raftNode.log, args.Entries...)
		}

		log.Printf("[%s] Appended %d entries from leader %s. New log size: %d", raftNode.cfg.NodeID, len(args.Entries), args.LeaderID, len(raftNode.log))
	}

	// commit Entries
	if args.LeaderCommitIndex > raftNode.commitIndex {
		newEntryLog := len(raftNode.log) - 1
		raftNode.commitIndex = min(args.LeaderCommitIndex, newEntryLog)
	}

	log.Printf("[%s] Heartbeat received from leader %s. Resetting election timer.", raftNode.cfg.NodeID, args.LeaderID)
	raftNode.resetElectionTimer()
	reply.Success = true

	return reply
}

func (raftNode *RaftNode) sendAppendEntries() {
	raftNode.mutex.Lock()
	defer raftNode.mutex.Unlock()

	// Update commit index based on majority match
	for n := len(raftNode.log) - 1; n > raftNode.commitIndex; n-- {
		count := 1
		for _, peer := range raftNode.cfg.Peers {
			if raftNode.matchIndex[peer] >= n {
				count++
			}
		}
		if count > (len(raftNode.cfg.Peers)+1)/2 {
			raftNode.commitIndex = n
			log.Printf("[%s] Set commit index to %d", raftNode.cfg.NodeID, n)
			break
		} else {
			log.Printf("[%s] Couldn't get majority to commit the log", raftNode.cfg.NodeID)
		}
	}

	for _, peerAddr := range raftNode.cfg.Peers {
		raftNode.mutex.Lock()
		nextIdx := raftNode.nextIndex[peerAddr]
		prevLogIndex := nextIdx - 1
		var prevLogTerm int
		if prevLogIndex >= 0 {
			prevLogTerm = raftNode.log[prevLogIndex].Term
		} else {
			prevLogTerm = 0
		}
		entries := append([]LogEntry{}, raftNode.log[nextIdx:]...)
		term := raftNode.currentTerm
		leaderID := raftNode.cfg.NodeID
		commitIdx := raftNode.commitIndex
		raftNode.mutex.Unlock()

		go func(addr string, nextIdx int, prevLogIndex int, prevLogTerm int, entries []LogEntry, term int, leaderID string, commitIdx int) {
			args := AppendEntriesArgs{
				Term:              term,
				LeaderID:          leaderID,
				PrevLogIndex:      prevLogIndex,
				PrevLogTerm:       prevLogTerm,
				Entries:           entries,
				LeaderCommitIndex: commitIdx,
			}

			url := fmt.Sprintf("http://%s/appendEntries", addr)
			jsonData, _ := json.Marshal(args)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				return
			}
			defer resp.Body.Close()

			var reply AppendEntriesReply
			if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
				return
			}

			raftNode.mutex.Lock()
			defer raftNode.mutex.Unlock()

			if reply.Term > raftNode.currentTerm {
				raftNode.becomeFollower(reply.Term)
				return
			}

			if raftNode.state == Leader && raftNode.currentTerm == term {
				if reply.Success {
					raftNode.nextIndex[addr] = prevLogIndex + 1 + len(entries)
					raftNode.matchIndex[addr] = raftNode.nextIndex[addr] - 1
				} else {
					raftNode.nextIndex[addr] = max(1, raftNode.nextIndex[addr]-1)
				}
			}
		}(peerAddr, nextIdx, prevLogIndex, prevLogTerm, entries, term, leaderID, commitIdx)
	}
}

func (raftNode *RaftNode) SubmitCommand(command Command) bool {
	raftNode.mutex.Lock()
	defer raftNode.mutex.Unlock()

	if raftNode.state != Leader {
		return false
	}

	entry := LogEntry{
		Term:    raftNode.currentTerm,
		Command: command,
	}
	raftNode.log = append(raftNode.log, entry)
	log.Printf("[%s] Appended new command to log: %+v", raftNode.cfg.NodeID, command)
	return true
}
