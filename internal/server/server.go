package server

import (
	"context"
	"fmt"
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

func (s *Server) Serve(ctx context.Context, fifo *fifo.Fifo) error {
	fmt.Printf("%s\n", fifo.Path())
	for {
		p, err := packet.ReadPacket(ctx, fifo)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Errorf("error parsing packet: %v", err)
		}
		s.handleCommand(ctx, p)
	}
}
