package main

import (
	"fmt"
	"os"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
	"path/filepath"
)

func serverFifoPath(dir string, pid int) string {
      return filepath.Join(dir, fmt.Sprintf("treenity.server.%d", pid))
}

func openServerFifo(mode fifo.Mode) (*fifo.Fifo, error) {
	pid := os.Getpid()

	path := serverFifoPath("/tmp", pid)
	return fifo.Create(path, mode)
}

func main() {
	f, err := openServerFifo(fifo.ReadWrite)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create fifo: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()
	
	fmt.Println(f.Path())
}
