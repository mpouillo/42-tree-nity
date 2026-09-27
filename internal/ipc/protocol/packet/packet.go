package packet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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

func ReadPacket(ctx context.Context, r io.Reader) (*Packet, error) {
	type result struct {
		packet *Packet
		err    error
	}

	ch := make(chan result, 1)

	go func() {
		var header PacketHeader
		if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
			ch <- result{nil, err}
			return
		}

		if header.Magic != MagicByte {
			ch <- result{nil, errors.New("invalid protocol magic byte")}
			return
		}

		if header.PayloadLength > MaxPayloadSize {
			ch <- result{nil, fmt.Errorf("payload length %d exceeds max allowed size %d", header.PayloadLength, MaxPayloadSize)}
			return
		}

		payload := make([]byte, header.PayloadLength)
		if _, err := io.ReadFull(r, payload); err != nil {
			ch <- result{nil, err}
			return
		}

		ch <- result{&Packet{Header: header, Payload: payload}, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.packet, res.err
	}
}

func WritePacket(w io.Writer, command uint8, payload []byte) error {
	if len(payload) > int(MaxPayloadSize) {
		return fmt.Errorf("payload size %d exceeds max allowed size %d", len(payload), MaxPayloadSize)
	}

	header := PacketHeader{
		Magic:         MagicByte,
		Command:       command,
		PayloadLength: uint16(len(payload)),
	}

	if err := binary.Write(w, binary.LittleEndian, &header); err != nil {
		return err
	}

	if len(payload) > 0 {
		n, err := w.Write(payload)
		if err != nil {
			return err
		}
		if n < len(payload) {
			return io.ErrShortWrite
		}
	}

	return nil
}
