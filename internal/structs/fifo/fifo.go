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
	owner     bool      // true = we create the fifo and must remove it
	closeOnce sync.Once // avoid closing multiple times
	closeErr  error
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
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false, fmt.Errorf("fifo: %s exists but it is not a FIFO (%s)", path, fi.Mode())
	}
	return false, nil
}

// Open opens a named FIFO at the specified path with the specified opening mode.
func Open(path string, openmode Mode) (*Fifo, error) {
	flags, err := openmode.flags()
	if err != nil {
		return nil, err
	}

	// O_NOFOLLOW: refuse symlinks (keep files in /tmp)
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

// Create creates a named FIFO at the specified path and opens it with the specified opening mode.
// If the FIFO already existed, it is reused and will not be removed on Close.
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

func (f *Fifo) Read(p []byte) (int, error)  { return f.file.Read(p) }
func (f *Fifo) Write(p []byte) (int, error) { return f.file.Write(p) }
func (f *Fifo) Path() string                { return f.path }

// Closes the Fifo and deletes the file if we are owner of it
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
