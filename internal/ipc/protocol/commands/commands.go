package commands

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

// in order to avoid a problem when 2 clients with the same name subscribe, we have two
// fifos for subscribe
type Subscribe struct {
	IPCPath      string `kong:"-"` // client checks if this is ok
	ConsumerPath string `kong:"-"` // then it checks here for messages
	Topic        string `arg:""`   // Topic before Client for kong arg order
	Client       string `arg:""`
	Prefix       string
	Offset       *uint32 `json:",omitempty"` // nil = no offset specified and else the pointer is used
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
