package commands

import "github.com/mpouillo/42-tree-nity/internal/message"

const (
	CmdNone        uint8 = 0
	CmdCreateTopic uint8 = 1
	CmdListTopics  uint8 = 2
	CmdInfoClient  uint8 = 3
	CmdProduce     uint8 = 4
	CmdSubscribe   uint8 = 5
	CmdAckOffset   uint8 = 6
	CmdDisconnect  uint8 = 7
)

type IPCPathProvider interface {
	GetIPCPath() string
}

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
	IPCPath string          `kong:"-"`
	Topic   string          `arg:""`
	Message message.Message `kong:"-"`
	Raw     bool            `json:"-"`
}

// two fifos, so a duplicate client never reads the messages of the first one
type Subscribe struct {
	IPCPath      string `kong:"-"` // client checks if this is ok
	ConsumerPath string `kong:"-"` // then it checks here for messages
	Topic        string `arg:""`   // Topic before Client for kong arg order
	Client       string `arg:""`
	Prefix       string
	Offset       *uint32 `json:",omitempty"` // nil when --offset is not given
	Raw          bool    `json:"-"`
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

func (c CreateTopic) GetIPCPath() string {
	return c.IPCPath
}

func (c ListTopics) GetIPCPath() string {
	return c.IPCPath
}

func (c InfoClient) GetIPCPath() string {
	return c.IPCPath
}

func (c Produce) GetIPCPath() string {
	return c.IPCPath
}

func (c Subscribe) GetIPCPath() string {
	return c.IPCPath
}

func (c AckOffset) GetIPCPath() string {
	return c.IPCPath
}

func (c Disconnect) GetIPCPath() string {
	return c.IPCPath
}
