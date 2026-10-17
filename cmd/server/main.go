package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/server"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
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
		log.Fatalf("failed to create fifo: %v\n", err)
	}
	defer func() { _ = f.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.NewServer()
	go srv.Serve(f)

	// wait for the consumers to commit before removing the fifo
	<-ctx.Done()
	srv.Shutdown()
	removeClientFifos(f.Path())
}

// removes the fifos left by the clients of this server: a killed client
// cannot remove its own
func removeClientFifos(serverPath string) {
	paths, err := filepath.Glob(serverPath + ".*")
	if err != nil {
		return
	}
	for _, p := range paths {
		fi, err := os.Lstat(p)
		if err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warnf("failed to remove %s: %v", p, err)
		}
	}
}
