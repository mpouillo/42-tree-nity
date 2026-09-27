package server

import (
	"encoding/json"

	"github.com/ayberkgezer/gocolorlog"
	commands "github.com/mpouillo/42-tree-nity/internal/ipc/protocol/command"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

func processPacket[T any](p *packet.Packet) (*T, error) {
	var obj T
	if err := json.Unmarshal(p.Payload, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func (s *Server) handleCommand(p *packet.Packet) {
	if p == nil {
		return
	}

	switch p.Header.Command {
	case commands.CmdCreateTopic:
		s.handleCreateTopic(p)
	case commands.CmdListTopics:
		s.handleListTopics(p)
	case commands.CmdProduce:
		s.handleProduce(p)
	case commands.CmdSubscribe:
		s.handleSubscribe(p)
	default:
		gocolorlog.Error("unknown command")
	}
}

func (s *Server) handleListTopics(p *packet.Packet) {
	payload, err := processPacket[commands.ListTopics](p)
	if err != nil {
		gocolorlog.Errorf("error processing packet payload: %v", err)
		return
	}

	outFifo, err := fifo.Create(payload.IPCPath, fifo.Write)
	if err != nil {
		gocolorlog.Errorf("error creating response fifo: %v", err)
		return
	}
	defer outFifo.Close()

	s.mu.RLock()
	topicList := make([]string, 0, len(s.topics))
	for name := range s.topics {
		topicList = append(topicList, name)
	}
	s.mu.RUnlock()

	dataBytes, _ := json.Marshal(response.ListTopicsData{Topics: topicList})

	resp := response.Response{
		Code: response.NoError,
		Data: dataBytes,
	}

	packet.WritePacket(outFifo, commands.CmdListTopics, resp)
}