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
    owner bool      // true = we create the fifo and must remove it
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
	// O_NOFOLLOW: refuse symlinks (keep files in /tmp)
	file, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		file.Close()
		return nil, fmt.Errorf("fifo: %s is not a FIFO (%s)", path, fi.Mode())
	}

	return &Fifo{path: path, file: file}, nil
}

// Create creates a named FIFO at the specified path and opens it with the specified flags.
func Create(path string, flag int) (*Fifo, error) {
	we_created, err := createFifo(path)
	if err != nil {
		return nil, err
	}
	fifo, err := Open(path, flag)
	if err != nil {
		if we_created{
			os.Remove(path)
		}
		return nil, err
	}
	fifo.owner = we_created
	return fifo, nil
}

func (f *Fifo) Read(p []byte) (int, error)  { return f.file.Read(p) }
func (f *Fifo) Write(p []byte) (int, error) { return f.file.Write(p) }

// Closes the Fifo and deletes the file if we are owner of it
func (fifo *Fifo) Close() error {
	fifo.closeOnce.Do(func() {
		if fifo.file != nil {
			fifo.closeErr = fifo.file.Close()
		}

		if fifo.owner {
			err := os.Remove(fifo.path) 
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if fifo.closeErr == nil {
					fifo.closeErr = err
				}
			}
		}
	})

	return fifo.closeErr
}