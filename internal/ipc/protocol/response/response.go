package response

import "encoding/json"

const (
	NoError      uint8 = 0
	GeneralError uint8 = 1
	TopicError   uint8 = 2
	IpcError     uint8 = 3
)

type ListTopicsData struct {
	Topics []string `json:"topics"`
}

type InfoClientData struct {
	Client string `json:"client"`
	Topic  string `json:"topic"`
	Offset uint32 `json:"offset"`
	Prefix string `json:"prefix"`
	IPC    string `json:"ipc"`
}

type SubscribeData struct {
	Message string `json:"message"`
}

type Response struct {
	Code     uint8           `json:"code"`
	ErrorMsg string          `json:"error,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}
