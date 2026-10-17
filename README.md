*This project has been created as part of the 42 curriculum by mpouillo, cobussie.*

# Tree-nity

- [Description](#description)
    - [Overview](#overview)
    - [Architecture](#architecture)
    - [IPC Choice](#ipc-choice)
    - [Data Structures](#data-structures)
    - [Testing](#testing)
- [Instructions](#instructions)
- [Contribution](#contribution)
- [Resources](#resources)

## Description

### Overview

Tree-nity is a small message queue system written in Go, inspired by Kafka. Producers publish messages to named topics, and consumers subscribe to these topics to receive the messages asynchronously, at their own pace.

The project is made of two programs that communicate through named pipes (FIFOs):

- `server`: stores topics and messages in memory, keeps track of every consumer and delivers the messages.
- `client`: a single executable that creates and lists topics, queries client metadata, produces messages read from stdin, or subscribes to a topic and prints the messages it receives.

Main features:

- **Topics** holding an ordered log of messages, each identified by a 32-bit offset.
- **Text and binary messages**: `key:body` lines, or length-prefixed records with `--raw` (up to 1024 bytes of key and body).
- **Prefix filtering**: a consumer can receive only the messages whose key starts with a given prefix.
- **Offset management**: each consumer commits the next offset it wants to read, so that it resumes where it stopped after a reconnection. It can also start from any offset with `--offset`.
- **Graceful shutdown**: on `SIGINT` or `SIGTERM`, the server waits up to 5 seconds for the consumers to commit, makes them exit with code 0 and removes every FIFO.

### Architecture

```
 producer / management clients           consumer clients
        │  create, list, info,            │  subscribe, ack,
        │  produce                        │  disconnect
        ▼                                 ▼
 ┌──────────── main FIFO /tmp/treenity.server.<pid> ─────────────┐
 │                                                               │
 │  Serve goroutine ── reads packets, runs the handlers          │
 │       │                                                       │
 │       ├── client index (custom hashmap)                       │
 │       │                                                       │
 │       └── topics ── one goroutine per topic                   │
 │                       ├── message log                         │
 │                       └── prefix trie of consumers            │
 └───────────────────────────────┬───────────────────────────────┘
          answers                │ messages
   <ipc>.@<pid> FIFOs            ▼ <ipc>.<client> FIFOs
```

**Goroutines.**

- The **Serve goroutine** reads the packets of the main FIFO one by one and runs the matching handler: create, list, info, produce, subscribe, ack and disconnect. Handlers are short and never block on a client, so a single reader keeps the packets in the order they arrived and needs no extra locking.
- Each topic runs **its own goroutine** (`Topic.run`), which owns message dispatching. It receives events on Go channels:
    - `msgChan`: a new message was produced, so the matching consumers are woken up;
    - `subChan`: a consumer subscribed, so it starts receiving from its offset;
    - `ackChan`: a consumer acknowledged its message, so it gets the next one;
    - a ticker that disconnects the consumers that did not ack within 3 seconds;
    - the context of the topic, cancelled on shutdown.
- Each consumer has **at most one message in flight**: the topic sends the next matching message only after the previous one was acknowledged. This gives back-pressure for free, and messages are always delivered in offset order.

**Synchronization.**

- `Server.mu` (`sync.RWMutex`) protects the topic map and the client index.
- `Topic.mu` (`sync.RWMutex`) protects the message log, the consumer map and the prefix trie. `Produce` takes the write lock to append a message, while lookups and dispatching take the read lock.
- The delivery state of a consumer (`InFlight`, `SentAt`) is only touched by the goroutine of its topic, so it needs no lock. Its committed offset is an `atomic.Uint32`, because the Serve goroutine writes it and the topic goroutine reads it.
- A `context.Context` per topic, derived from the server, stops the topic goroutine. A `sync.WaitGroup` waits for it to finish.

**Shutdown.** On `SIGINT` or `SIGTERM`, the server closes all the topics in parallel:

1. Each topic stops dispatching and waits up to 5 seconds for the in-flight acks.
2. It sends a disconnect packet with exit code 0 to each of its consumers, then closes their FIFOs.
3. The server removes its main FIFO and every client FIFO left in `/tmp`.

The Serve goroutine keeps reading during the shutdown, so the last acks are still committed.

### IPC Choice

We chose **named pipes (FIFOs)**.

- **Standard library support:** they are supported by the Go standard library (`syscall.Mkfifo`, `os.OpenFile`), so neither CGo nor an external library is needed.
- **Identifiable endpoints:** a FIFO is a file, so the IPC identifier is simply its path (`/tmp/treenity.server.<pid>`). The PID makes it unique, so several servers can run side by side, and cleaning up only means removing files.
- **Atomic writes:** writes of at most `PIPE_BUF` (4096 bytes on Linux) bytes are atomic. Our packets are capped at 4096 bytes, so the packets of concurrent clients never interleave on the shared main FIFO.
- **Detecting a missing server:** opening a FIFO for writing in non-blocking mode fails when nobody reads it, which is how the client detects that the server is not running.

FIFOs also have limitations, which we had to work around:

- **One-way:** a FIFO has no reply channel.
- **Plain byte stream:** there are no message boundaries. We frame every message as a packet: a 4-byte header (magic byte `0x42`, command, payload length as a little-endian `uint16`) followed by a JSON payload. The reader checks the header and the JSON before consuming anything, and skips invalid bytes to resynchronize on the next valid packet.
- **No sender identity:** a reader cannot tell who wrote the data, so each request carries the information needed to answer it.

**Bidirectional communication.** Every client creates its own FIFOs and puts their path in its requests:

- **Commands** (create, list, info, produce): the client creates `<ipc>.@<pid>` and sends its path with the request. The server writes the answer to it.
- **Consumers:** they receive their messages on a dedicated FIFO named after the client, `<ipc>.<client>`; this is the `ipc` path reported by `info`. A separate PID FIFO is used for the subscription handshake, so that a second client using the same name never reads the messages of the first one. Acks and disconnects go to the main FIFO.
- **Shutdown:** closing a one-way FIFO does not wake up a blocked reader, so the server sends an explicit disconnect packet carrying exit code 0 before closing the consumer FIFO. If the server dies, the consumer reads EOF and exits with code 3.

### Data structures

**Client metadata: custom hashmap.** The client index (`internal/structs/hashmap`) is a generic hashmap written by hand.

- **Hashing:** 64-bit FNV-1a, modulo the number of buckets.
- **Collisions:** separate chaining; each bucket holds a slice of key-value entries.
- **Growth:** the map starts with 16 buckets and doubles once the load factor reaches 0.75, rehashing every entry. Inserting an existing key replaces its value without growing the map.
- **Content:** the client name maps to its metadata: topic, committed offset, prefix, consumer FIFO path and PID. Entries are never deleted, so a returning consumer resumes from its committed offset.

**Prefix matching: trie of consumers.** Each topic indexes its consumers in a trie (`internal/structs/trie`), keyed by their prefix. A consumer is stored at the node where its prefix ends, and several consumers can share a node.

- **Search:** to find the consumers of a message, we walk down the trie along the message key and collect the consumers of every node visited. These are exactly the consumers whose prefix is a prefix of the key.
- **Cost:** a search is O(length of the key), no matter how many consumers there are. Comparing the key with every prefix would be O(number of consumers). A hash table of prefixes would require one lookup per prefix length of the key.
- **Empty prefix:** consumers without a prefix are not stored in the trie. They are added directly to the delivery list, since they match every message.
- **Message log:** messages are stored in a slice where the index of a message is its offset. A consumer that catches up resumes directly at its committed offset, without searching.

The topic map itself is a regular Go map: the subject only requires a custom hashmap for the client metadata.

### Testing

Unit tests use Go's `testing` package, with [testify](https://github.com/stretchr/testify) for assertions. Run them all with:

```sh
make test
```

This builds every package, then runs `go test ./...`; the exit code is non-zero if a test fails.

The prefix-matching tests are in two places:

- `internal/structs/trie/trie_test.go` tests the trie itself:
    - insertion and removal;
    - exact, partial and non-matching searches, and overlapping prefixes;
    - the empty-prefix wildcard and several consumers sharing a prefix;
    - an empty trie;
    - a large-scale test: 1000 prefixes, checked against a brute-force scan.
- `internal/topic/topic_test.go` tests the complete filtering:
    - `TestMatchConsumers` mixes wildcard and prefix consumers;
    - `TestPrefixFiltering_Integration` subscribes consumers with different kinds of prefixes over real FIFOs. It checks exactly which messages each one receives, and in which order, both for stored messages and for new ones.

To run a single package or test, or to use the race detector:

```sh
go test -v ./internal/structs/trie/
go test -v -run TestPrefixFiltering_Integration ./internal/topic/
go test -race ./...
```

## Instructions

**Requirements:** Linux and Go 1.27.1. If Go is not installed, `make install-go` installs it, along with golangci-lint, in your home directory.

> **Warning:** `make install-go` deletes `~/go` and appends the Go environment to `~/.profile`.

**Build:**

```sh
make        # builds ./server and ./client
make re     # rebuilds from scratch
make clean  # removes the binaries and the downloaded attachments
make lint   # runs golangci-lint
```

**Run the server.** It prints the IPC identifier to give to the clients:

```sh
./server
/tmp/treenity.server.1337
```

**Use the client:**

```sh
./client <ipc> create <topic>
./client <ipc> list
./client <ipc> info <client>
./client <ipc> produce <topic> [--raw]
./client <ipc> subscribe <topic> <client> [--prefix <prefix>] [--offset <offset>] [--raw]
```

Topic and client names must match `^[a-zA-Z0-9_.-]{1,32}$`.

**Example:**

```sh
./client /tmp/treenity.server.1337 create user_events
./client /tmp/treenity.server.1337 subscribe user_events client0 --prefix user.create &
printf 'user.create:{"user":"reach"}\nuser.delete:{"user":"froz"}\n' \
    | ./client /tmp/treenity.server.1337 produce user_events
./client /tmp/treenity.server.1337 info client0
```

**Message formats:**

- **Text (default):** one `key:body` message per line; the body is everything after the first colon. Empty lines are skipped. A non-empty line without a colon is an error.
- **Raw (`--raw`):**
    - producer input: `[keysize:int32][key][valuesize:int32][value]`;
    - consumer output: `[offset:int32][keysize:int32][key][valuesize:int32][value]`.

    All integers are little-endian, and records are written back to back.

**Exit codes:**

| Code | Meaning |
|---|---|
| 0 | Success, or graceful stop on `SIGINT`/`SIGTERM` |
| 1 | General error: invalid arguments or names, server unreachable, invalid input |
| 2 | Topic error: unknown or duplicate topic, unknown client, name already in use |
| 3 | IPC error: the server disconnected or stopped answering |

When several errors apply, the first one in the order 1, then 3, then 2 is returned.

**Helper binaries.** The subject provides `fish` (a producer) and `aquarium` (a consumer visualizer). Extract the subject attachments at the root of the repository so that `jammy/fish` and `jammy/aquarium` exist, then run:

```sh
make bubble
```

This starts a server, creates a topic, launches 20 fish producers and opens the aquarium.

## Contribution

[<img src="https://contrib.rocks/image?repo=mpouillo/42-tree-nity">](https://github.com/mpouillo/42-tree-nity/graphs/contributors)

- **mpouillo**:
	- Trie
	- Server
	- Topic, message, consumer, commands
- **cobussie**:
	- Hashmap
	- Client
	- Fifo
	- Response structure


## Resources

- [fifo(7)](https://man7.org/linux/man-pages/man7/fifo.7.html), [mkfifo(3)](https://man7.org/linux/man-pages/man3/mkfifo.3.html) and [pipe(7)](https://man7.org/linux/man-pages/man7/pipe.7.html): FIFOs, non-blocking opens and `PIPE_BUF` atomicity
- [Go documentation](https://pkg.go.dev/std): `os`, `syscall`, `sync`, `sync/atomic`, `context`, `os/signal`
- [Trie (Wikipedia)](https://en.wikipedia.org/wiki/Trie)
- [Implementing tries in Go](https://rlayte.github.io/blog/2015/05/10/implementing-tries-in-go/)
- [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests)
- [Hash table: separate chaining (Wikipedia)](https://en.wikipedia.org/wiki/Hash_table#Separate_chaining) and [FNV hash (Wikipedia)](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function)
- [Conventional Commits](https://www.conventionalcommits.org/)

### AI usage

AI was used throughout the project:

- to learn about and understand the various topics necessary to building this project;
- to review the project against the subject and find issues;
- to write part of the tests;
- to rewrite the function documentation comments and to draft this README.

Every change was reviewed, tested and understood by the team before being merged.
