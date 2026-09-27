package packet

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteAndReadPacket(t *testing.T) {
	tests := []struct {
		name    string
		command uint8
		payload []byte
	}{
		{
			name:    "empty payload",
			command: 1,
			payload: []byte{},
		},
		{
			name:    "regular payload",
			command: 2,
			payload: []byte("hello world"),
		},
		{
			name:    "max payload size",
			command: 3,
			payload: bytes.Repeat([]byte("a"), int(MaxPayloadSize)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WritePacket(&buf, tt.command, tt.payload)
			assert.NoError(t, err)

			got, err := ReadPacket(context.Background(), &buf)
			assert.NoError(t, err)

			want := &Packet{
				Header: PacketHeader{
					Magic:         MagicByte,
					Command:       tt.command,
					PayloadLength: uint16(len(tt.payload)),
				},
				Payload: tt.payload,
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestWritePacketErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "exceeds max payload size",
			payload: bytes.Repeat([]byte("a"), int(MaxPayloadSize)+1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WritePacket(&buf, 1, tt.payload)
			assert.Error(t, err)
		})
	}
}

func TestReadPacketValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{
			name: "invalid magic byte",
			raw:  []byte{0x00, 0x01, 0x04, 0x00, 't', 'e', 's', 't'},
		},
		{
			name: "payload length exceeds max size",
			raw:  []byte{MagicByte, 0x01, 0xFF, 0xFF},
		},
		{
			name: "incomplete header",
			raw:  []byte{MagicByte, 0x01},
		},
		{
			name: "incomplete payload",
			raw:  []byte{MagicByte, 0x01, 0x10, 0x00, 'a', 'b'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewReader(tt.raw)

			got, err := ReadPacket(context.Background(), buf)
			assert.Error(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestReadPacketContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r, _ := io.Pipe()

	got, err := ReadPacket(ctx, r)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, got)
}
