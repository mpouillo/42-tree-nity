package consumer

import (
	"fmt"
	"regexp"
	"sync/atomic"

	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	message "github.com/mpouillo/42-tree-nity/internal/message"
	fifo "github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

type Consumer struct {
	ID     string
	Topic  string
	Prefix string
	Offset atomic.Uint32 // next offset to consume
	Fifo   *fifo.Fifo

	InFlight bool // a message is sent but not acknoleged
}

func NewConsumer(id, topic, prefix string, offset uint32, ipcPath string) (*Consumer, error) {
	if !validName.MatchString(id) {
		return nil, fmt.Errorf("invalid name %q (expected %s)", id, validName)
	}

	fifo, err := fifo.Open(ipcPath, fifo.Write)
	if err != nil {
		return nil, err
	}

	c := &Consumer{
		ID:     id,
		Topic:  topic,
		Prefix: prefix,
		Fifo:   fifo,
	}
	c.Offset.Store(offset)
	return c, nil
}

func (c *Consumer) Deliver(msg message.TopicMessage) error {
	if c == nil || c.Fifo == nil {
		return nil
	}

	data := response.SubscribeData{Message: msg}
	return packet.WriteResponse(c.Fifo, commands.CmdSubscribe, response.NoError, "", data)
}

func (c *Consumer) CloseIPCChannel() error {
	if c == nil || c.Fifo == nil {
		return nil
	}

	return c.Fifo.Close()
}
