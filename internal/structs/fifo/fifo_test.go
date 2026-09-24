package fifo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPath(t *testing.T) {
	path := "/tmp/test.fifo"
	f := &Fifo{path: path}

	if f.Path() != path {
		t.Fatalf("expected path %q, got %q", path, f.Path())
	}
}

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

		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = f.Close() }()

		if f.Path() != path {
			t.Fatalf("expected path %q, got %q", path, f.Path())
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

		f, err := Create(path, ReadWrite)
		if err == nil {
			t.Fatalf("expected error for non-fifo path, got nil")
		}
		if f != nil {
			t.Fatalf("expected nil Fifo, got %v", f)
		}
	})

	t.Run("create with an unknown mode removes the fifo it created", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")

		f, err := Create(path, Mode(42))
		if err == nil {
			_ = f.Close()
			t.Fatalf("expected error for unknown mode, got nil")
		}

		_, statErr := os.Lstat(path)
		if !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("expected fifo to be removed, got err %v", statErr)
		}
	})

	t.Run("create on existing fifo is not owner", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		if _, err := createFifo(path); err != nil {
			t.Fatalf("setup: %v", err)
		}

		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = f.Close() }()

		if f.owner {
			t.Fatalf("expected owner = false when the fifo already existed")
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

		f, err := Open(path, ReadWrite)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = f.Close() }()

		if f.Path() != path {
			t.Fatalf("expected path %q, got %q", path, f.Path())
		}
	})

	t.Run("open nonexistent path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.fifo")

		_, err := Open(path, Read)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("open symlink to fifo is rejected", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.fifo")
		_, err := createFifo(target)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		link := filepath.Join(dir, "link.fifo")
		err = os.Symlink(target, link)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Open(link, ReadWrite)
		if err == nil {
			_ = f.Close()
			t.Fatalf("expected error for symlink, got nil")
		}
	})

	t.Run("open with an unknown mode is rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		_, err := createFifo(path)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Open(path, Mode(42))
		if err == nil {
			_ = f.Close()
			t.Fatalf("expected error for unknown mode, got nil")
		}
	})

	t.Run("open regular file is rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "regular")
		err := os.WriteFile(path, []byte("not a fifo"), 0600)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Open(path, ReadWrite)
		if err == nil {
			_ = f.Close()
			t.Fatalf("expected error for regular file, got nil")
		}
	})
}

func TestRead(t *testing.T) {
	t.Run("read data written to fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}
		defer func() { _ = f.Close() }()

		want := []byte("hello")
		_, err = f.Write(want)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		got := make([]byte, len(want))
		n, err := f.Read(got)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != len(want) {
			t.Fatalf("expected to read %d bytes, got %d", len(want), n)
		}
		if string(got) != string(want) {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("read from closed fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		_, err = f.Read(make([]byte, 1))
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestWrite(t *testing.T) {
	t.Run("write data to fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}
		defer func() { _ = f.Close() }()

		data := []byte("hello")
		n, err := f.Write(data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != len(data) {
			t.Fatalf("expected to write %d bytes, got %d", len(data), n)
		}
	})

	t.Run("write to closed fifo", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		_, err = f.Write([]byte("hello"))
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("close removes fifo when owner", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, statErr := os.Lstat(path)
		if !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("expected fifo to be removed, got err %v", statErr)
		}
	})

	t.Run("close keeps fifo when not owner", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		_, err := createFifo(path)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		f, err := Open(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatalf("expected fifo to still exist, got err %v", statErr)
		}
	})

	t.Run("close can be called multiple times", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		first := f.Close()
		second := f.Close()
		if first != nil {
			t.Fatalf("expected no error on first close, got %v", first)
		}
		if second != first {
			t.Fatalf("expected same result on second close, got %v vs %v", second, first)
		}
	})

	t.Run("close returns the same error on repeated calls", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.fifo")
		f, err := Create(path, ReadWrite)
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		err = f.file.Close()
		if err != nil {
			t.Fatalf("setup: expected no error, got %v", err)
		}

		first := f.Close()
		second := f.Close()
		if first == nil {
			t.Fatalf("expected an error on first close, got nil")
		}
		if second != first {
			t.Fatalf("expected same error on second close, got %v vs %v", second, first)
		}
	})
}
