package fifo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	t.Run("check fifo created", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		created, err := Create(path)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !created {
			t.Fatalf("expected created = true, got false")
		}
		fi, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatalf("expected fifo file to exist, got error: %v", statErr)
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("expected created file to be a fifo, got mode %s", fi.Mode())
		}
	})

	t.Run("check fifo already exists", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		if _, err := Create(path); err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		created, err := Create(path)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if created {
			t.Fatalf("expected created = false, got true")
		}
	})

	t.Run("check path exists but is not a fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		if err := os.WriteFile(path, []byte("not a fifo"), 0600); err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		created, err := Create(path)
		if err == nil {
			t.Fatalf("expected error for non-fifo path, got nil")
		}
		if created {
			t.Fatalf("expected created = false, got true")
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("open existing fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		if _, err := Create(path); err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Open(path, os.O_RDWR)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer f.file.Close()

		if f.path != path {
			t.Fatalf("expected path %q, got %q", path, f.path)
		}
	})

	t.Run("open nonexistent path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.fifo")

		_, err := Open(path, os.O_RDONLY)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}
