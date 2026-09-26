package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrInvalidMessage = errors.New("error parsing message from raw byte value")

type Message struct {
	Key    string
	Body   []byte
	Offset uint32
}

func FromSeparator(msg []byte, sep string) *Message {
	key, body, found := bytes.Cut(msg, []byte(sep))

	if !found {
		body = []byte{}
	} else {
		body = bytes.Clone(body)
	}

	return &Message{
		Key:    string(key),
		Body:   body,
		Offset: 0,
	}
}

func FromRaw(msg []byte) (*Message, error) {
	offset := 0

	if len(msg)-offset < 4 {
		return nil, fmt.Errorf("%w: missing key size header", ErrInvalidMessage)
	}
	keysize := int(binary.LittleEndian.Uint32(msg[offset : offset+4]))
	offset += 4

	if len(msg)-offset < keysize {
		return nil, fmt.Errorf("%w: truncated key bytes", ErrInvalidMessage)
	}
	key := string(msg[offset : offset+keysize])
	offset += keysize

	if len(msg)-offset < 4 {
		return nil, fmt.Errorf("%w: missing body size header", ErrInvalidMessage)
	}
	bodysize := int(binary.LittleEndian.Uint32(msg[offset : offset+4]))
	offset += 4

	if len(msg)-offset < bodysize {
		return nil, fmt.Errorf("%w: truncated body bytes", ErrInvalidMessage)
	}
	body := bytes.Clone(msg[offset : offset+bodysize])

	return &Message{
		Key:    key,
		Body:   body,
		Offset: 0,
	}, nil
}

func (m *Message) ToRaw() []byte {
	buf := make([]byte, 0, 4+len(m.Key)+4+len(m.Body))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Key)))
	buf = append(buf, m.Key...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Body)))
	buf = append(buf, m.Body...)
	return buf
}