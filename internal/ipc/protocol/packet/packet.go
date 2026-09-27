package packet

import (
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

func ReadPacket(r io.Reader) (*Packet, error) {
	var header PacketHeader

	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Magic != MagicByte {
		return nil, errors.New("invalid protocol magic byte")
	}

	if header.PayloadLength > MaxPayloadSize {
		return nil, fmt.Errorf("payload length %d exceeds max allowed size", header.PayloadLength)
	}

	payload := make([]byte, header.PayloadLength)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	return &Packet{header, payload}, nil
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
