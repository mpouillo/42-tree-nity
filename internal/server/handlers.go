package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
	"github.com/mpouillo/42-tree-nity/internal/topic"
)

var (
	ErrGeneralError   = errors.New("internal server error")
	ErrDuplicateTopic = errors.New("topic already exists")
)

func (s *Server) handleCommand(ctx context.Context, p *packet.Packet) {
	if p == nil {
		return
	}

	switch p.Header.Command {
	case commands.CmdCreateTopic:
		s.handleCreateTopic(ctx, p)
	case commands.CmdListTopics:
		s.handleListTopics(p)
	case commands.CmdInfoClient:
		s.handleInfoClient(p)
	case commands.CmdProduce:
		s.handleProduce(p)
	case commands.CmdSubscribe:
		s.handleSubscribe(p)
	case commands.CmdAckOffset:
		s.handleAckOffset(p)
	case commands.CmdDisconnect:
		s.handleDisconnect(p)
	default:
		log.Error("unknown command")
	}
}

func (s *Server) handleCreateTopic(ctx context.Context, p *packet.Packet) {
	f := func(payload *commands.CreateTopic) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.topics[payload.Topic]; exists {
			return nil, ErrDuplicateTopic
		}

		newTopic, err := topic.NewTopic(ctx, payload.Topic)
		if err != nil {
			return nil, err
		}

		newTopic.OnDisconnect = func(clientID string) {
			client, err := s.clients.Get(clientID)
			if err != nil || client == nil {
				return
			}

			s.sendError(client.Fifo, commands.CmdDisconnect, response.IpcError, "")
		}
		s.topics[newTopic.Name] = newTopic

		return []byte("topic created"), nil
	}

	s.executeWithResponse(p, commands.CmdCreateTopic, f)
}

func (s *Server) handleListTopics(p *packet.Packet) {
	f := func(payload *commands.ListTopics) (any, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		topicList := make([]string, 0, len(s.topics))
		for name := range s.topics {
			topicList = append(topicList, name)
		}

		return response.ListTopicsData{Topics: topicList}, nil
	}

	s.executeWithResponse(p, commands.CmdListTopics, f)
}

func (s *Server) handleInfoClient(p *packet.Packet) {
	f := func(payload *commands.InfoClient) (any, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		clientInfo, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, fmt.Errorf("error getting client info: %v", err)
		}

		data := response.InfoClientData{
			Client: clientInfo.ID,
			Topic:  clientInfo.Topic,
			Prefix: clientInfo.Prefix,
			Offset: clientInfo.Offset.Load(),
			IPC:    clientInfo.Fifo.Path(),
		}

		return data, nil
	}

	s.executeWithResponse(p, commands.CmdInfoClient, f)
}

func (s *Server) handleProduce(p *packet.Packet) {
	f := func(payload *commands.Produce) (any, error) {
		_, err := s.topics[payload.Topic].Produce(payload.Message.Key, payload.Message.Body)
		if err != nil {
			return nil, fmt.Errorf("error producing message: %v", err)
		}

		return []byte("message produced successfully"), nil
	}

	s.executeWithResponse(p, commands.CmdProduce, f)
}

func (s *Server) handleSubscribe(p *packet.Packet) {
	f := func(payload *commands.Subscribe) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		topic := s.topics[payload.Topic]

		client, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, fmt.Errorf("error getting consumer data from database: %v", err)
		}
		if client == nil {
			client, err = consumer.NewConsumer(payload.Client, payload.Topic, payload.Prefix, *payload.Offset, payload.ConsumerPath)
			if err != nil {
				return nil, fmt.Errorf("error creating consumer: %v", err)
			}
		}

		if err := topic.Subscribe(client); err != nil {
			return nil, fmt.Errorf("error subscribing consumer: %v", err)
		}

		s.clients.Insert(client.ID, client)
		return []byte("subscription successful"), nil
	}

	s.executeWithResponse(p, commands.CmdSubscribe, f)
}

func (s *Server) handleAckOffset(p *packet.Packet) {
	f := func(payload *commands.AckOffset) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		client, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, fmt.Errorf("error getting consumer data from database: %v", err)
		}

		client.Offset.Store(payload.Offset)

		topic := s.topics[client.Topic]
		if topic != nil {
			currentOffset := client.Offset.Load()
			if int(currentOffset) < len(topic.GetMessages()) {
				if err := client.Deliver(topic.GetMessages()[currentOffset]); err != nil {
					return nil, err
				}
			}
		}

		return []byte("offset acknowleged successfully"), nil
	}

	s.executeWithResponse(p, commands.CmdAckOffset, f)
}

func (s *Server) handleDisconnect(p *packet.Packet) {
	f := func(payload *commands.Disconnect) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		client, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, fmt.Errorf("error getting consumer list: %v", err)
		}

		topic := s.topics[client.Topic]
		if topic == nil {
			return nil, nil
		}

		_ = client.CloseIPCChannel()
		topic.Unsubscribe(client.ID)
		return []byte("disconnected successfully"), nil
	}

	s.executeWithResponse(p, commands.CmdDisconnect, f)
}

func (s *Server) executeWithResponse[T commands.IPCPathProvider](p *packet.Packet, cmd uint8, f func(payload *T) (any, error)) {
	// unmarshall packet payload
	payload, err := processPacket[T](p)
	if err != nil {
		log.Errorf("error processing packet payload: %v", err)
		return
	}

	// open response FIFO
	outFifo, err := fifo.Open(payload.GetIPCPath(), fifo.Write)
	if err != nil {
		log.Errorf("error creating response fifo: %v", err)
		return
	}

	// run function
	data, err := f(&payload)

	// handle errors
	if err != nil {
		e := err.Error()
		log.Error(e)

		errorCode := response.GeneralError
		if errors.Is(err, ErrDuplicateTopic) || errors.Is(err, topic.ErrTopicClosed) {
			errorCode = response.TopicError
		}
		s.sendError(outFifo, cmd, errorCode, e)
		return
	}

	// handle data
	if err := s.sendSuccess(outFifo, cmd, data); err != nil {
		log.Errorf("error sending response: %v", err)
	}
}

func processPacket[T any](p *packet.Packet) (T, error) {
	var obj T
	if err := json.Unmarshal(p.Payload, &obj); err != nil {
		return obj, err
	}
	return obj, nil
}

func (s *Server) sendSuccess(w io.Writer, cmd uint8, data any) error {
	var dataBytes []byte
	var err error

	if data != nil {
		dataBytes, err = json.Marshal(data)
		if err != nil {
			s.sendError(w, cmd, response.GeneralError, ErrGeneralError.Error())
			return err
		}
	}

	r := response.Response{
		Code: response.NoError,
		Data: dataBytes,
	}

	rBytes, err := json.Marshal(r)
	if err != nil {
		s.sendError(w, cmd, response.GeneralError, ErrGeneralError.Error())
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
