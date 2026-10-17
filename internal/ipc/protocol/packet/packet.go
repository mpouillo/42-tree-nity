package packet

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
)

const (
	MagicByte      uint8  = 0x42
	HeaderSize            = 4
	MaxPayloadSize uint16 = 4096 - HeaderSize
)

type PacketHeader struct {
	Magic         uint8
	Command       uint8
	PayloadLength uint16
}

type Packet struct {
	Header  PacketHeader
	Payload []byte
}

// NewReader returns a reader that can hold a whole packet. Create only one per
// FIFO, as it buffers the bytes read in advance.
func NewReader(r io.Reader) *bufio.Reader {
	return bufio.NewReaderSize(r, HeaderSize+int(MaxPayloadSize))
}

// ReadPacket returns the next valid packet read from r, or ctx.Err() if ctx
// is done first.
func ReadPacket(ctx context.Context, r *bufio.Reader) (*Packet, error) {
	type result struct {
		packet *Packet
		err    error
	}

	ch := make(chan result, 1)

	go func() {
		p, err := readValidPacket(r)
		ch <- result{p, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.packet, res.err
	}
}

// readValidPacket returns the next valid packet of r, skipping invalid bytes one
// by one. Nothing is consumed before the whole packet has been checked.
func readValidPacket(r *bufio.Reader) (*Packet, error) {
	for {
		raw, err := r.Peek(HeaderSize)
		if err != nil {
			return nil, err
		}
		header := PacketHeader{
			Magic:         raw[0],
			Command:       raw[1],
			PayloadLength: binary.LittleEndian.Uint16(raw[2:]),
		}
		if !validHeader(header) {
			_, _ = r.Discard(1)
			continue
		}

		raw, err = r.Peek(HeaderSize + int(header.PayloadLength))
		if errors.Is(err, io.EOF) {
			_, _ = r.Discard(1)
			continue
		}
		if err != nil {
			return nil, err
		}
		if !json.Valid(raw[HeaderSize:]) {
			_, _ = r.Discard(1)
			continue
		}

		payload := bytes.Clone(raw[HeaderSize:])
		_, _ = r.Discard(len(raw))
		return &Packet{Header: header, Payload: payload}, nil
	}
}

// validHeader reports whether h has the magic byte, a known command and an allowed payload length.
func validHeader(h PacketHeader) bool {
	return h.Magic == MagicByte &&
		h.Command != commands.CmdNone && h.Command <= commands.CmdDisconnect &&
		h.PayloadLength <= MaxPayloadSize
}

// WritePacket writes a command packet with payload to w in a single write, so
// that it stays atomic on a FIFO.
func WritePacket(w io.Writer, command uint8, payload []byte) error {
	if len(payload) > int(MaxPayloadSize) {
		return fmt.Errorf("payload size %d exceeds max allowed size %d", len(payload), MaxPayloadSize)
	}

	buf := make([]byte, 0, HeaderSize+len(payload))
	buf = append(buf, MagicByte, command)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(payload)))
	buf = append(buf, payload...)

	n, err := w.Write(buf)
	if err != nil {
		return err
	}
	if n < len(buf) {
		return io.ErrShortWrite
	}
	return nil
}

// WriteResponse writes a cmd packet holding a response with code, errorMsg and
// data encoded as JSON.
func WriteResponse(w io.Writer, cmd uint8, code uint8, errorMsg string, data any) error {
	var dataBytes []byte
	var err error

	if data != nil {

		switch v := data.(type) {
		case json.RawMessage:
			dataBytes = v
		default:
			dataBytes, err = json.Marshal(data)
			if err != nil {
				return err
			}
		}
	}

	res := response.Response{
		Code:     code,
		ErrorMsg: errorMsg,
		Data:     dataBytes,
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return WritePacket(w, cmd, resBytes)
}
