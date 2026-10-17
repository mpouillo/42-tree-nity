package packet

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndReadPacket(t *testing.T) {
	tests := []struct {
		name    string
		command uint8
		payload []byte
	}{
		{
			name:    "empty object",
			command: 1,
			payload: []byte(`{}`),
		},
		{
			name:    "regular payload",
			command: 2,
			payload: []byte(`"hello world"`),
		},
		{
			name:    "max payload size",
			command: 3,
			payload: []byte(`"` + strings.Repeat("a", int(MaxPayloadSize)-2) + `"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WritePacket(&buf, tt.command, tt.payload)
			assert.NoError(t, err)

			got, err := ReadPacket(context.Background(), NewReader(&buf))
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
			got, err := ReadPacket(context.Background(), NewReader(bytes.NewReader(tt.raw)))
			assert.ErrorIs(t, err, io.EOF)
			assert.Nil(t, got)
		})
	}
}

func TestReadPacketContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r, _ := io.Pipe()

	got, err := ReadPacket(ctx, NewReader(r))
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, got)
}

// returns the bytes of a packet and the packet ReadPacket should give back
func encode(t *testing.T, command uint8, payload string) ([]byte, *Packet) {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, WritePacket(&buf, command, []byte(payload)))

	return buf.Bytes(), &Packet{
		Header: PacketHeader{
			Magic:         MagicByte,
			Command:       command,
			PayloadLength: uint16(len(payload)),
		},
		Payload: []byte(payload),
	}
}

func TestReadPacketResync(t *testing.T) {
	first, firstPacket := encode(t, 1, `{"Topic":"events"}`)
	second, secondPacket := encode(t, 2, `{"IPCPath":"/tmp/x"}`)

	tests := []struct {
		name string
		raw  []byte
		want []*Packet
	}{
		{
			name: "back-to-back packets",
			raw:  slices.Concat(first, second),
			want: []*Packet{firstPacket, secondPacket},
		},
		{
			name: "garbage before a packet",
			raw:  slices.Concat([]byte("garbage!"), first),
			want: []*Packet{firstPacket},
		},
		{
			name: "garbage containing the magic byte",
			raw:  slices.Concat([]byte("BONJOUR"), first, second),
			want: []*Packet{firstPacket, secondPacket},
		},
		{
			name: "fake header eating the start of the next packet",
			raw:  slices.Concat([]byte{MagicByte, 1, 2, 0}, first),
			want: []*Packet{firstPacket},
		},
		{
			name: "plausible header with a payload that is not JSON",
			raw:  slices.Concat([]byte{MagicByte, 1, 3, 0, 'a', 'b', 'c'}, first),
			want: []*Packet{firstPacket},
		},
		{
			name: "unknown command",
			raw:  slices.Concat([]byte{MagicByte, 99, 2, 0, '{', '}'}, first),
			want: []*Packet{firstPacket},
		},
		{
			name: "fake header announcing more bytes than the stream has",
			raw:  slices.Concat([]byte{MagicByte, 1, 0x00, 0x0F}, first),
			want: []*Packet{firstPacket},
		},
		{
			name: "garbage only",
			raw:  []byte("garbage!"),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(bytes.NewReader(tt.raw))

			for _, want := range tt.want {
				got, err := ReadPacket(context.Background(), r)
				require.NoError(t, err)
				assert.Equal(t, want, got)
			}

			got, err := ReadPacket(context.Background(), r)
			assert.ErrorIs(t, err, io.EOF)
			assert.Nil(t, got)
		})
	}
}
