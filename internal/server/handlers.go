package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
	"github.com/mpouillo/42-tree-nity/internal/structs/hashmap"
	"github.com/mpouillo/42-tree-nity/internal/topic"
)

var (
	ErrGeneralError   = errors.New("internal server error")
	ErrDuplicateTopic = errors.New("topic already exists")

	// errDuplicateClient aliases topic.ErrDuplicateClient for handlers that shadow the topic package.
	errDuplicateClient = topic.ErrDuplicateClient
)

// handleCommand dispatches packet p to the handler of its command.
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
		log.Errorf("unknown command: %v", p.Header.Command)
	}
}

// handleCreateTopic creates a topic and starts its goroutine, or fails if the name is taken.
func (s *Server) handleCreateTopic(ctx context.Context, p *packet.Packet) {
	f := func(payload *commands.CreateTopic) (any, uint8, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.topics[payload.Topic]; exists {
			return nil, response.TopicError, ErrDuplicateTopic
		}

		newTopic, err := topic.NewTopic(ctx, payload.Topic)
		if err != nil {
			return nil, response.TopicError, err
		}

		newTopic.OnDisconnect = func(c *consumer.Consumer, code uint8) {
			s.sendError(c.Fifo, commands.CmdDisconnect, code, "server connection closed")
		}
		s.topics[newTopic.Name] = newTopic

		log.Infof("created topic: %v", newTopic.Name)
		return []byte("topic created"), response.NoError, nil
	}

	s.executeWithResponse(p, commands.CmdCreateTopic, f)
}

// handleListTopics answers with the names of all the topics.
func (s *Server) handleListTopics(p *packet.Packet) {
	f := func(payload *commands.ListTopics) (any, uint8, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		topicList := make([]string, 0, len(s.topics))
		for name := range s.topics {
			topicList = append(topicList, name)
		}

		log.Infof("retrieved topic list: %v", topicList)
		return response.ListTopicsData{Topics: topicList}, response.NoError, nil
	}

	s.executeWithResponse(p, commands.CmdListTopics, f)
}

// handleInfoClient answers with the metadata stored for a client.
func (s *Server) handleInfoClient(p *packet.Packet) {
	f := func(payload *commands.InfoClient) (any, uint8, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		clientInfo, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, response.TopicError, fmt.Errorf("error getting client info: %v", err)
		}

		data := response.InfoClientData{
			Client: clientInfo.ID,
			Topic:  clientInfo.Topic,
			Prefix: clientInfo.Prefix,
			Offset: clientInfo.Offset.Load(),
			IPC:    clientInfo.Fifo.Path(),
		}

		log.Infof("retrieved client info: %v", data)
		return data, response.NoError, nil
	}

	s.executeWithResponse(p, commands.CmdInfoClient, f)
}

// handleProduce stores a message in its topic, which delivers it to the matching consumers.
func (s *Server) handleProduce(p *packet.Packet) {
	f := func(payload *commands.Produce) (any, uint8, error) {
		t, exists := s.topics[payload.Topic]
		if !exists {
			return nil, response.TopicError, fmt.Errorf("topic does not exist")
		}

		_, err := t.Produce(payload.Message.Key, payload.Message.Body)
		if err != nil {
			if err == topic.ErrTopicClosed {
				return nil, response.IpcError, fmt.Errorf("error producing message: %v", err)
			}
			return nil, response.TopicError, fmt.Errorf("error producing message: %v", err)
		}

		log.Infof("produced message: %v", payload.Message)
		return []byte("message produced successfully"), response.NoError, nil
	}

	s.executeWithResponse(p, commands.CmdProduce, f)
}

// handleSubscribe registers a consumer, resuming from its committed offset unless
// another is given. A dead previous client with the same name is replaced.
func (s *Server) handleSubscribe(p *packet.Packet) {
	f := func(payload *commands.Subscribe) (any, uint8, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		t, exists := s.topics[payload.Topic]
		if !exists {
			return nil, response.TopicError, fmt.Errorf("topic does not exist")
		}

		previous, err := s.clients.Get(payload.Client)
		if err != nil && err != hashmap.ErrKeyNotFound {
			return nil, response.GeneralError, fmt.Errorf("error getting consumer data from database: %v", err)
		}

		var offset uint32
		if err == nil {
			if s.isConnected(previous) {
				if syscall.Kill(previous.PID, 0) == nil {
					return nil, response.TopicError, fmt.Errorf("error subscribing consumer: %v", errDuplicateClient)
				}
				s.topics[previous.Topic].Unsubscribe(previous.ID)
			}
			if previous.Topic == payload.Topic {
				offset = previous.Offset.Load()
			}
		}
		if payload.Offset != nil {
			offset = *payload.Offset
		}

		client, err := consumer.NewConsumer(payload.Client, payload.Topic, payload.Prefix, offset, payload.ConsumerPath)
		if err != nil {
			return nil, response.GeneralError, fmt.Errorf("error creating consumer: %v", err)
		}
		client.PID = payload.PID

		if err := t.Subscribe(client); err != nil {
			_ = client.CloseIPCChannel()
			if err == topic.ErrTopicClosed {
				return nil, response.IpcError, fmt.Errorf("error producing message: %v", err)
			}
			return nil, response.TopicError, fmt.Errorf("error subscribing consumer: %v", err)
		}
		_ = previous.CloseIPCChannel()

		s.clients.Insert(client.ID, client)

		log.Infof("subscribed consumer %v to topic %v", client.ID, t.Name)
		return []byte("subscription successful"), response.NoError, nil
	}

	s.executeWithResponse(p, commands.CmdSubscribe, f)
}

// handleAckOffset commits the offset acknowledged by a consumer and lets its
// topic send the next message.
func (s *Server) handleAckOffset(p *packet.Packet) {
	f := func(payload *commands.AckOffset) (any, uint8, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		client, err := s.clients.Get(payload.Client)
		if err != nil && err != hashmap.ErrKeyNotFound {
			return nil, response.TopicError, fmt.Errorf("error getting consumer data from database: %v", err)
		} else if err == hashmap.ErrKeyNotFound {
			return nil, response.TopicError, fmt.Errorf("client does not exist: %v", err)
		}

		topic := s.topics[client.Topic]
		if topic == nil {
			return nil, response.TopicError, fmt.Errorf("topic does not exist")
		}

		client.Offset.Store(payload.Offset)
		if err := topic.Ack(client); err != nil {
			return nil, response.TopicError, err
		}

		log.Infof("updated offset for consumer %v: %v", client.ID, client.Offset.Load())
		return []byte("offset acknowleged successfully"), response.NoError, nil
	}

	executeWithoutResponse(p, f)
}

// handleDisconnect unsubscribes a consumer that is exiting; its metadata is kept.
func (s *Server) handleDisconnect(p *packet.Packet) {
	f := func(payload *commands.Disconnect) (any, uint8, error) {
		s.mu.Lock()
		defer s.mu.Unlock()

		client, err := s.clients.Get(payload.Client)
		if err != nil {
			return nil, response.GeneralError, fmt.Errorf("error getting consumer list: %v", err)
		}

		topic := s.topics[client.Topic]
		if topic == nil {
			return nil, response.TopicError, fmt.Errorf("topic does not exist")
		}

		_ = client.CloseIPCChannel()
		topic.Unsubscribe(client.ID)

		log.Infof("unsubscribed consumer %v from topic %v", client.ID, topic.Name)
		return []byte("disconnected successfully"), response.NoError, nil
	}

	executeWithoutResponse(p, f)
}

// executeWithResponse decodes the payload of p, runs f on it and writes the
// result or the error to the response FIFO of the client.
func (s *Server) executeWithResponse[T commands.IPCPathProvider](p *packet.Packet, cmd uint8, f func(payload *T) (any, uint8, error)) {
	payload, err := processPacket[T](p)
	if err != nil {
		log.Errorf("error processing packet payload: %v", err)
		return
	}

	outFifo, err := fifo.Open(payload.GetIPCPath(), fifo.Write)
	if err != nil {
		log.Warnf("error creating response fifo: %v", err)
		return
	}
	defer func() { _ = outFifo.Close() }()

	data, code, err := f(&payload)
	if err != nil {
		e := err.Error()
		log.Error(e)
		s.sendError(outFifo, cmd, code, e)
		return
	}

	if err := s.sendSuccess(outFifo, cmd, data); err != nil {
		log.Errorf("error sending response: %v", err)
	}
}

// executeWithoutResponse decodes the payload of p and runs f on it, logging any
// error. It is used for the commands that get no answer (ack and disconnect).
func executeWithoutResponse[T any](p *packet.Packet, f func(payload *T) (any, uint8, error)) {
	payload, err := processPacket[T](p)
	if err != nil {
		log.Errorf("error processing packet payload: %v", err)
		return
	}

	if _, _, err := f(&payload); err != nil {
		log.Error(err.Error())
	}
}

// isConnected reports whether c is currently subscribed to its topic.
func (s *Server) isConnected(c *consumer.Consumer) bool {
	t := s.topics[c.Topic]
	return t != nil && t.HasConsumer(c.ID)
}

// processPacket decodes the JSON payload of p into a value of type T.
func processPacket[T any](p *packet.Packet) (T, error) {
	var obj T
	if err := json.Unmarshal(p.Payload, &obj); err != nil {
		return obj, err
	}
	return obj, nil
}

// sendSuccess writes a successful cmd response carrying data to w. It does not
// log w, whose FIFO may be closed concurrently.
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

	log.Debugf("packet written: %v", r)
	return packet.WritePacket(w, cmd, rBytes)
}

// sendError writes a cmd response with the error code and message msg to w.
func (s *Server) sendError(w io.Writer, cmd uint8, code uint8, msg string) {
	r := response.Response{
		Code:     code,
		ErrorMsg: msg,
	}
	rBytes, _ := json.Marshal(r)
	_ = packet.WritePacket(w, cmd, rBytes)
	log.Debugf("packet written: %v", r)
}
