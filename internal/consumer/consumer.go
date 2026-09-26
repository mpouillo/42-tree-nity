package consumer

import (
	message "github.com/mpouillo/42-tree-nity/internal/message"
	fifo "github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type Consumer struct {
	ID     string
	Topic  string
	Offset uint32
	Prefix string
	Fifo   *fifo.Fifo
}

func NewConsumer(id, ipcPath, topic, prefix string, offset uint32) (*Consumer, error) {
	fifo, err := fifo.Open(ipcPath, fifo.Write)

	if err != nil {
		return nil, err
	}

	return &Consumer{
		ID: id,
		Topic: topic,
		Offset: offset,
		Prefix: prefix,
		Fifo: fifo,
	}, nil
}

func (c *Consumer) Deliver(msg message.Message) {}

func (c *Consumer) CloseIPCChannel() {}
