package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
)

const (
	CmdNone        uint8 = 0 // No Command
	CmdCreateTopic uint8 = 1 // Manager: Create topic
	CmdListTopics  uint8 = 2 // Manager: List topics
	CmdInfoClient  uint8 = 3 // Manager: Query client metadata
	CmdProduce     uint8 = 4 // Producer: Handshake / verify topic existence
	CmdSubscribe   uint8 = 5 // Consumer: Subscribe & register dedicated FIFO
	CmdAckOffset   uint8 = 6 // Consumer: Acknowledge committed offset
	CmdDisconnect  uint8 = 7 // Client: Disconnect notification
)

type CreateTopic struct {
	IPCPath string `kong:"-"`
	Topic   string `arg:""`
}

type ListTopics struct {
	IPCPath string `kong:"-"`
}

type InfoClient struct {
	IPCPath string `kong:"-"`
	Client  string `arg:""`
}

type Produce struct {
	IPCPath string `kong:"-"`
	Topic   string `arg:""`
	Message []byte `kong:"-"`
	Raw     bool   `json:"-"`
}

type Subscribe struct {
	IPCPath string `kong:"-"`
	Topic   string `arg:""` // Topic before Client for kong arg order
	Client  string `arg:""`
	Prefix  string
	Offset  *uint32 `json:",omitempty"` // nil = no offset specified and else the pointer is used
	Raw     bool    `json:"-"`
}

type AckOffset struct {
	IPCPath string
	Client  string
	Offset  uint32
}

type Disconnect struct {
	IPCPath string
	Client  string
}

func NewProduce(packet packet.Packet) (*Produce, error) {
	if packet.Header.Command != CmdProduce {
		return nil, errors.New("invalid packet command type")
	}
	var p Produce
	if err := json.Unmarshal(packet.Payload, &p); err != nil {
		return nil, err
	}
	if length := len(p.Message); length > 1024 {
		return nil, fmt.Errorf("packet length %d exceeds max allowed size", length)
	}

	return &p, nil
}
