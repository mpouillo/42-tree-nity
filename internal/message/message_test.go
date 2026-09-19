package message

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromSeparator(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		key, body string
	}{
		{
			name:  "empty message",
			input: "",
			key:   "", body: "",
		},
		{
			name:  "regular message",
			input: "user.input:hello, world!",
			key:   "user.input", body: "hello, world!",
		},
		{
			name:  "multiple separators",
			input: "user.input:hello:world!",
			key:   "user.input", body: "hello:world!",
		},
		{
			name:  "only separator",
			input: ":",
			key:   "", body: "",
		},
		{
			name:  "no separator",
			input: "user.input",
			key:   "user.input", body: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FromSeparator([]byte(tt.input), ":")
			expected := &Message{Key: tt.key, Body: []byte(tt.body), Offset: 0}

			assert.Equal(t, expected, actual)
		})
	}
}

func TestFromRaw(t *testing.T) {
	tests := []struct {
		name string
		key  string
		body string
	}{
		{
			name: "empty message",
			key:  "",
			body: "",
		},
		{
			name: "regular message",
			key:  "user.input",
			body: "hello, world!",
		},
		{
			name: "empty key with body",
			key:  "",
			body: "hello world",
		},
		{
			name: "key with empty body",
			key:  "user.input",
			body: "",
		},
		{
			name: "raw binary body",
			key:  "metrics",
			body: string([]byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := encodeMessage(tt.key, tt.body)

			actual, err := FromRaw(msg)
			expected := &Message{Key: tt.key, Body: []byte(tt.body), Offset: 0}

			assert.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestFromRaw_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty buffer",
			input: []byte{},
		},
		{
			name:  "truncated key size header",
			input: []byte{0x01, 0x00, 0x00}, // 3 bytes instead of 4
		},
		{
			name:  "key size exceeds buffer length",
			input: []byte{0x0A, 0x00, 0x00, 0x00, 'a', 'b'}, // declares 10-byte key, provides 2
		},
		{
			name:  "truncated body size header",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x01}, // valid 0-byte key, truncated body header
		},
		{
			name:  "body size exceeds buffer length",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 'a'}, // declares 5-byte body, provides 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromRaw(tt.input)

			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidMessage)
		})
	}
}

func TestToRaw(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "empty message",
			msg:  &Message{Key: "", Body: []byte(""), Offset: 0},
		},
		{
			name: "regular message",
			msg:  &Message{Key: "user.input", Body: []byte("hello, world!"), Offset: 0},
		},
		{
			name: "empty key with body",
			msg:  &Message{Key: "", Body: []byte("body payload"), Offset: 0},
		},
		{
			name: "key with empty body",
			msg:  &Message{Key: "user.input", Body: []byte(""), Offset: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualBytes := tt.msg.ToRaw()
			expectedBytes := encodeMessage(tt.msg.Key, string(tt.msg.Body))

			assert.Equal(t, expectedBytes, actualBytes)

			decodedMsg, err := FromRaw(actualBytes)
			assert.NoError(t, err)
			assert.Equal(t, tt.msg, decodedMsg)
		})
	}
}

func encodeMessage(key, body string) []byte {
	buf := make([]byte, 0, 4+len(key)+4+len(body))

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)))
	buf = append(buf, body...)

	return buf
}