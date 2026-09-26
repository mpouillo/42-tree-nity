package consumer

import (
	message "github.com/mpouillo/42-tree-nity/internal/message"
	fifo "github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type Consumer struct {
	ID      string
	Topic   string
	Offset  uint32
	Prefix  string
	ipcPath fifo.Fifo
}

func (c *Consumer) Deliver(msg message.Message) {}

func (c *Consumer) CloseIPCChannel() {}
