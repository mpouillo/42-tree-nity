package fifo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFifo(t *testing.T) {
	t.Run("check fifo created", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		created, err := createFifo(path)
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
		_, err := createFifo(path)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		created, err := createFifo(path)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if created {
			t.Fatalf("expected created = false, got true")
		}
	})

	t.Run("check path exists but is not a fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		err := os.WriteFile(path, []byte("not a fifo"), 0600)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		created, err := createFifo(path)
		if err == nil {
			t.Fatalf("expected error for non-fifo path, got nil")
		}
		if created {
			t.Fatalf("expected created = false, got true")
		}
	})
}

func TestCreate(t *testing.T) {
	t.Run("create new fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")

		f, err := Create(path, os.O_RDWR)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer f.file.Close()

		if f.path != path {
			t.Fatalf("expected path %q, got %q", path, f.path)
		}
		if !f.owner {
			t.Fatalf("expected owner = true, got false")
		}

		fi, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatalf("expected fifo file to exist, got error: %v", statErr)
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("expected created file to be a fifo, got mode %s", fi.Mode())
		}
	})

	t.Run("create fails when path exists but is not a fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		err := os.WriteFile(path, []byte("not a fifo"), 0600)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Create(path, os.O_RDWR)
		if err == nil {
			t.Fatalf("expected error for non-fifo path, got nil")
		}
		if f != nil {
			t.Fatalf("expected nil Fifo, got %v", f)
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("open existing fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		_, err := createFifo(path)
		if err != nil {
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
