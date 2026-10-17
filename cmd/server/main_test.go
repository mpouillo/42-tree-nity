package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

// TestServerFifoPath checks that the server FIFO path is built from the PID.
func TestServerFifoPath(t *testing.T) {
	t.Run("normal check", func(t *testing.T) {
		expected := "/tmp/treenity.server.50"
		actual := serverFifoPath("/tmp", 50)
		if actual != expected {
			t.Errorf("expected %s, got %s", expected, actual)
		}
	})
	t.Run("check different output", func(t *testing.T) {
		expected := "/tmp/treenity.server.1120"
		actual := serverFifoPath("/tmp", 1120)
		if actual != expected {
			t.Errorf("expected %s, got %s", expected, actual)
		}
	})
}

// TestOpenServerFifo checks that the server creates a FIFO at its PID-based path.
func TestOpenServerFifo(t *testing.T) {
	f, err := openServerFifo(fifo.ReadWrite)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func() { _ = f.Close() }()

	want := serverFifoPath("/tmp", os.Getpid())
	if f.Path() != want {
		t.Fatalf("expected path %q, got %q", want, f.Path())
	}

	fi, err := os.Lstat(want)
	if err != nil {
		t.Fatalf("expected fifo to exist, got %v", err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("expected a fifo, got mode %s", fi.Mode())
	}
}

// TestRemoveClientFifos checks that only the client FIFOs of this server are
// removed, not its own FIFO, regular files or the FIFOs of another server.
func TestRemoveClientFifos(t *testing.T) {
	dir := t.TempDir()
	srv := serverFifoPath(dir, 50)

	mkfifo := func(path string) {
		t.Helper()
		if err := syscall.Mkfifo(path, 0600); err != nil {
			t.Fatalf("mkfifo %s: %v", path, err)
		}
	}
	mkfifo(srv)
	mkfifo(srv + ".client0")
	mkfifo(srv + ".@42")
	mkfifo(filepath.Join(dir, "treenity.server.500.c"))
	if err := os.WriteFile(srv+".txt", nil, 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	removeClientFifos(srv)

	for path, kept := range map[string]bool{
		srv:              true,
		srv + ".client0": false,
		srv + ".@42":     false,
		srv + ".txt":     true,
		filepath.Join(dir, "treenity.server.500.c"): true,
	} {
		_, err := os.Lstat(path)
		if exists := err == nil; exists != kept {
			t.Errorf("%s: expected kept=%v, got exists=%v", path, kept, exists)
		}
	}
}
