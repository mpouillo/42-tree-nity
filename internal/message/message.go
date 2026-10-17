package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// MaxSize is the maximum size of key and body together, in bytes.
const MaxSize = 1024

var ErrInvalidMessage = errors.New("error parsing message from raw byte value")
var ErrTooLarge = fmt.Errorf("message exceeds %d bytes (key+body)", MaxSize)

type Message struct {
	Key  []byte
	Body []byte
}

type TopicMessage struct {
	Message
	Offset uint32
}

// FromRaw decodes the single raw message held by msg.
func FromRaw(msg []byte) (*Message, error) {
	m, err := ReadRaw(bytes.NewReader(msg))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	return m, nil
}

// ReadRaw reads one [keysize:int32][key][valuesize:int32][value] message from r.
// It returns io.EOF only if r ends before the message starts.
func ReadRaw(r io.Reader) (*Message, error) {
	var size [4]byte

	if _, err := io.ReadFull(r, size[:]); err != nil {
		return nil, err
	}
	keySize := binary.LittleEndian.Uint32(size[:])
	if keySize > MaxSize {
		return nil, ErrTooLarge
	}
	key := make([]byte, keySize)
	if err := readFull(r, key); err != nil {
		return nil, err
	}

	if err := readFull(r, size[:]); err != nil {
		return nil, err
	}
	bodySize := binary.LittleEndian.Uint32(size[:])
	if bodySize > MaxSize-keySize {
		return nil, ErrTooLarge
	}
	body := make([]byte, bodySize)
	if err := readFull(r, body); err != nil {
		return nil, err
	}

	return &Message{
		Key:  key,
		Body: body,
	}, nil
}

// readFull is like io.ReadFull but returns io.ErrUnexpectedEOF instead of io.EOF.
func readFull(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// FromSeparator splits msg at the first sep into key and body. Without sep, the
// whole msg is the key and the body is empty.
func FromSeparator(msg []byte, sep string) *Message {
	key, body, found := bytes.Cut(msg, []byte(sep))

	if !found {
		body = []byte{}
	} else {
		body = bytes.Clone(body)
	}
	key = bytes.Clone(key)

	return &Message{
		Key:  key,
		Body: body,
	}
}

// ToRaw encodes m as [offset:int32][keysize:int32][key][valuesize:int32][value].
func (m *TopicMessage) ToRaw() []byte {
	buf := make([]byte, 0, 4+4+len(m.Key)+4+len(m.Body))
	buf = binary.LittleEndian.AppendUint32(buf, m.Offset)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Key)))
	buf = append(buf, m.Key...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Body)))
	buf = append(buf, m.Body...)
	return buf
}

// ToSeparator returns key, sep and body joined together.
func (m *Message) ToSeparator(sep string) []byte {
	buf := make([]byte, 0, len(m.Key)+len(sep)+len(m.Body))
	buf = append(buf, m.Key...)
	buf = append(buf, sep...)
	buf = append(buf, m.Body...)
	return buf
}
