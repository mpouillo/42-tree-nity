package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// max size of key+body
const MaxSize = 1024

var ErrInvalidMessage = errors.New("error parsing message from raw byte value")
var ErrTooLarge = fmt.Errorf("message exceeds %d bytes (key+body)", MaxSize)

type Message struct {
	Key    string
	Body   []byte
	Offset uint32
}

func FromRaw(msg []byte) (*Message, error) {
	m, err := ReadRaw(bytes.NewReader(msg))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	return m, nil
}

// reads one message [keysize:int32][key][valuesize:int32][value]
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
		Key:    string(key),
		Body:   body,
		Offset: 0,
	}, nil
}

// does the same as io.readFull but an EOF is considered an error
func readFull(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
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

func (m *Message) ToRaw() []byte {
	buf := make([]byte, 0, 4+4+len(m.Key)+4+len(m.Body))
	buf = binary.LittleEndian.AppendUint32(buf, m.Offset)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Key)))
	buf = append(buf, m.Key...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Body)))
	buf = append(buf, m.Body...)
	return buf
}

func (m *Message) ToSeparator(sep string) []byte {
	buf := make([]byte, 0, len(m.Key)+len(sep)+len(m.Body))
	buf = append(buf, m.Key...)
	buf = append(buf, sep...)
	buf = append(buf, m.Body...)
	return buf
}
