package fifo

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
)

type Fifo struct {
	path      string
	file      *os.File
	owner     bool // true = we create the fifo and must remove it
	closeOnce sync.Once
	closeErr  error
}

// createFifo creates a FIFO at path and reports whether it was created; an
// existing FIFO is accepted, any other existing file (even a symlink) is not.
func createFifo(path string) (created bool, err error) {
	err = syscall.Mkfifo(path, fifoPerms)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, syscall.EEXIST) {
		return false, err
	}

	fi, statErr := os.Lstat(path)
	if statErr != nil {
		return false, statErr
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false, fmt.Errorf("fifo: %s exists but it is not a FIFO (%s)", path, fi.Mode())
	}
	return false, nil
}

// Open opens the existing FIFO at path with openmode. It refuses symlinks and
// files that are not FIFOs.
func Open(path string, openmode Mode) (*Fifo, error) {
	flags, err := openmode.flags()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("fifo: %s is not a FIFO (%s)", path, fi.Mode())
	}

	return &Fifo{path: path, file: file}, nil
}

// Create creates the FIFO at path and opens it with openmode. An existing FIFO
// is reused, and is then not removed by Close.
func Create(path string, openmode Mode) (*Fifo, error) {
	WeCreated, err := createFifo(path)
	if err != nil {
		return nil, err
	}

	f, err := Open(path, openmode)
	if err != nil {
		if WeCreated {
			_ = os.Remove(path)
		}
		return nil, err
	}
	f.owner = WeCreated
	return f, nil
}

// Read reads up to len(p) bytes from the FIFO.
func (f *Fifo) Read(p []byte) (int, error) { return f.file.Read(p) }

// Write writes p to the FIFO.
func (f *Fifo) Write(p []byte) (int, error) { return f.file.Write(p) }

// Path returns the path of the FIFO.
func (f *Fifo) Path() string { return f.path }

// Close closes the FIFO and removes it if it was created by Create. Only the
// first call has an effect.
func (f *Fifo) Close() error {
	f.closeOnce.Do(func() {
		f.closeErr = f.file.Close()

		if f.owner {
			err := os.Remove(f.path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if f.closeErr == nil {
					f.closeErr = err
				}
			}
		}
	})

	return f.closeErr
}
