# YAML (YAML Aint Markup Language)
It is a document like markup language which is a human readable data serialization language.

# Go routines
go routines ensure no waiter starvation condition happens, the highest priority is writer lock. hence, when reader lock is first arrived but having writer lock in the queuew waiter will be evaluated.

## Duplex
duplex refers to transfering signls and data flows

### Full Duplex
- Data flows on both directions at same time
- Ex: Ethernet, mobile communication

### Half Duplex
- Data flows on both directions but only 1 device at a time
- Ex: Walkie-Talkie

### Multiplexer
The device that allows multiple signal and send one selected input to the output.


# Go Channels
- Channels are conduit which through go routines can sends or receive data.
- Channels can be created in specific datatype. ex: `chan int, chan string, chan struct etc..`

### UnBuffered Channel
 - It has no capacity, when data recieves that is sent to other goroutines
 - Sender blocks (waits) until a receiver is ready.
 - Receiver blocks until a sender sends something.
 - Synchronus

### Buffered Channel
- It has capacity and receives data until it gets full and data will be recieved until it gets empty
- Asynchronous

### Go leak
- When a goroutine is waiting for sending but no reciever or waiting for reciving but no sender results in `go leak`. `select` with `default` or `time.After` for non blocking helps.

## CSP Model (Communication sequential Processes)
- Go's concurrency comes from combining goroutines and channels.This is uses CSP, where different processes communicate through message passing(independing send & recieve) instead of shared memory access.


# RAFT
    - It is a consensus algorithm used in distributed systems to ensure every servers agrees to same state
    - RAFT = Reliable and Understandable and Consisten protocol