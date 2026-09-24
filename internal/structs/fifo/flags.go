package fifo

import (
	"fmt"
	"os"
	"syscall"
)

type Mode int

const (
	Read Mode = iota
	Write
	ReadWrite
)

func (m Mode) flags() (int, error) {
	switch m {
	case Read:
		return os.O_RDONLY, nil
	case Write:
		return os.O_WRONLY | syscall.O_NONBLOCK, nil
	case ReadWrite:
		return os.O_RDWR, nil
	default:
		return 0, fmt.Errorf("unknown mode: %d", int(m))
	}
}
