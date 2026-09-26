package consumer

import (
	"encoding/binary"

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
		ID:     id,
		Topic:  topic,
		Offset: offset,
		Prefix: prefix,
		Fifo:   fifo,
	}, nil
}

func (c *Consumer) Deliver(msg message.Message) (int, error) {
	var contents []byte

	if msg.Raw {
		binary.LittleEndian.AppendUint32(contents, msg.Offset)
		contents = append(msg.ToRaw(), contents...)
	} else {
		contents = msg.ToSeparator(":")
	}

	return c.Fifo.Write(contents)
}

func (c *Consumer) CloseIPCChannel() error {
	return c.Fifo.Close()
}
