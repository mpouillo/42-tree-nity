package consumer

import (
	"sync/atomic"

	message "github.com/mpouillo/42-tree-nity/internal/message"
	fifo "github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type Consumer struct {
	ID     string
	Topic  string
	Prefix string
	Offset atomic.Uint32
	Fifo   *fifo.Fifo
}

func NewConsumer(id, topic, prefix string, offset uint32, ipcPath string) (*Consumer, error) {
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

func (c *Consumer) Deliver(msg message.TopicMessage) (int, error) {
	if c == nil || c.Fifo == nil {
		return 0, nil
	}

	return c.Fifo.Write(msg.ToRaw())
}

func (c *Consumer) CloseIPCChannel() error {
	if c == nil || c.Fifo == nil {
		return nil
	}

	return c.Fifo.Close()
}
