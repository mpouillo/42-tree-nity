package main

import (
	"os"
	"testing"
	"io"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

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


func TestServerMain(t *testing.T) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("setup: expected no error, got %v", err)
	}
	originalStdout := os.Stdout
	os.Stdout = stdoutWriter

	main()

	os.Stdout = originalStdout
	_ = stdoutWriter.Close()
	output, _ := io.ReadAll(stdoutReader)

	expectedOutput := serverFifoPath("/tmp", os.Getpid()) + "\n"
	if string(output) != expectedOutput {
		t.Fatalf("expected output %q, got %q", expectedOutput, output)
	}
}
