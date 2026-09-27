package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ayberkgezer/gocolorlog"

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
		gocolorlog.Fatalf("failed to create fifo: %v\n", err)
	}
	defer f.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.NewServer()

	if err := srv.Serve(ctx, f); err != nil {
		gocolorlog.Fatalf("server error: %v", err)
	}
}
