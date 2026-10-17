package server

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
	"github.com/mpouillo/42-tree-nity/internal/structs/hashmap"
	"github.com/mpouillo/42-tree-nity/internal/topic"

	"github.com/mpouillo/42-tree-nity/internal/consumer"
)

type Server struct {
	mu      sync.RWMutex
	topics  map[string]*topic.Topic
	clients *hashmap.Hashmap[*consumer.Consumer]
}

func NewServer() *Server {
	return &Server{
		topics:  make(map[string]*topic.Topic),
		clients: hashmap.NewHashMap[*consumer.Consumer](),
	}
}

func (s *Server) Serve(fifo *fifo.Fifo) {
	fmt.Printf("%s\n", fifo.Path())
	reader := packet.NewReader(fifo)
	for {
		// context.background receive acks when the server disconnects
		p, err := packet.ReadPacket(context.Background(), reader)
		// main closed the fifo after Shutdown: the server is exiting
		if errors.Is(err, os.ErrClosed) {
			return
		}
		if err != nil {
			log.Errorf("error parsing packet: %v", err)
		}
		s.handleCommand(context.Background(), p)
	}
}

// closes every topic
func (s *Server) Shutdown() {
	s.mu.RLock()
	topics := slices.Collect(maps.Values(s.topics))
	s.mu.RUnlock()

	var wg sync.WaitGroup
	for _, t := range topics {
		wg.Go(func() { _ = t.Close() })
	}
	wg.Wait()
}
