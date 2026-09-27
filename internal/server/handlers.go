package server

import (
	"encoding/json"
	"io"

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

	if err := s.sendSuccess(outFifo, commands.CmdListTopics, response.ListTopicsData{Topics: topicList}); err != nil {
		gocolorlog.Errorf("error sending list topics response: %v", err)
	}
}

func (s *Server) sendSuccess(w io.Writer, cmd uint8, data any) error {
	var dataBytes []byte
	var err error

	if data != nil {
		dataBytes, err = json.Marshal(data)
		if err != nil {
			s.sendError(w, cmd, response.GeneralError, "internal server error")
			return err
		}
	}

	r := response.Response{
		Code: response.NoError,
		Data: dataBytes,
	}

	rBytes, err := json.Marshal(r)
	if err != nil {
		s.sendError(w, cmd, response.GeneralError, "internal server error")
		return err
	}

	return packet.WritePacket(w, cmd, rBytes)
}

func (s *Server) sendError(w io.Writer, cmd uint8, code uint8, msg string) {
	r := response.Response{
		Code:     code,
		ErrorMsg: msg,
	}
	rBytes, _ := json.Marshal(r)
	_ = packet.WritePacket(w, cmd, rBytes)
}