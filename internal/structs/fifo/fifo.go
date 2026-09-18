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
    closeErr	error
}

// Create a named FIFO at the specified path.
// returns true if the FIFO was created, false if it already exists, and an error if any.
func createFifo(path string) (created bool, err error) {
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

func Create(path string, flag int) (*Fifo, error) {
	_, err := createFifo(path)
	if err != nil {
		return nil, err
	}
	f, err := Open(path, flag)
	if err != nil {
		os.Remove(path)
		return nil, err
	}
	f.owner = true
	return f, nil
}

func (f *Fifo) Read(p []byte) (int, error)  { return f.file.Read(p) }
func (f *Fifo) Write(p []byte) (int, error) { return f.file.Write(p) }

func (f *Fifo) Close() error {
	f.closeOnce.Do(func() {
		if f.file != nil {
			f.closeErr = f.file.Close()
		}

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