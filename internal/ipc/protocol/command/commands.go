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
	IPCPath string
	Topic   string
}

type ListTopics struct {
	IPCPath string
}

type InfoClient struct {
	IPCPath string
	Client  string
}

type Produce struct {
	IPCPath string
	Topic   string
	Message string
	Raw     bool
}

type Subscribe struct {
	IPCPath string
	Client  string
	Topic   string
	Prefix  string
	Offset  uint32
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
	json.Unmarshal(packet.Payload, &p)
	if length := len(p.Message); length > 1024 {
		return nil, fmt.Errorf("packet length %d exceeds max allowed size", length)
	}

	return &p, nil
}
