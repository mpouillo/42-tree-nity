package fifo

import (
    "os"
    "sync"
    "fmt"
    "syscall"
    "errors"
	)

type Fifo struct {
    path  string
    file  *os.File
    owner bool      // true = send, false = receive
    closeOnce  sync.Once // avoid closing multiple times
}

// Create a named FIFO at the specified path.
// returns true if the FIFO was created, false if it already exists, and an error if any.
func Create(path string) (created bool, err error) {
	err = syscall.Mkfifo(path, fifoPerms)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, syscall.EEXIST) {
		return false, err
	}

	// Lstat does not follow symlink
	fi, statErr := os.Lstat(path)
	if statErr != nil {
		return false, statErr
	}
	if fi.Mode() & os.ModeNamedPipe == 0 {
		return false, fmt.Errorf("fifo: %s exists but it is not a FIFO (%s)", path, fi.Mode())
	}
	return false, nil
}

// waits until the FIFO is available for the specified flags and then opens it.
func Open(path string, flags int) (*Fifo, error) {
	file, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, err
	}
	return &Fifo{path: path, file: file}, nil
}