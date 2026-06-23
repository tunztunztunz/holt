package provision

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCopy(t *testing.T) {
	t.Run("copies a file, preserving contents and mode", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dst := filepath.Join(dir, "dst")
		if err := os.WriteFile(src, []byte("hi"), 0o600); err != nil {
			t.Fatal(err)
		}

		skip, err := Copy(src, dst)
		if err != nil || skip != "" {
			t.Fatalf("Copy = (%q, %v), want ('', nil)", skip, err)
		}
		if got, _ := os.ReadFile(dst); string(got) != "hi" {
			t.Errorf("dst contents = %q, want %q", got, "hi")
		}
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("dst mode = %o, want 600", got)
		}
	})

	t.Run("missing source is skipped, not an error", func(t *testing.T) {
		dir := t.TempDir()
		dst := filepath.Join(dir, "dst")

		skip, err := Copy(filepath.Join(dir, "nope"), dst)
		if err != nil || skip != "source missing" {
			t.Fatalf("Copy = (%q, %v), want ('source missing', nil)", skip, err)
		}
		if _, err = os.Lstat(dst); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("dst should not exist, Lstat err = %v", err)
		}
	})

	t.Run("existing dest is left untouched", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dst := filepath.Join(dir, "dst")
		if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte("orig"), 0o600); err != nil {
			t.Fatal(err)
		}

		skip, err := Copy(src, dst)
		if err != nil || skip != "dest exists" {
			t.Fatalf("Copy = (%q, %v), want ('dest exists', nil)", skip, err)
		}
		if got, _ := os.ReadFile(dst); string(got) != "orig" {
			t.Errorf("dst overwritten: %q, want %q", got, "orig")
		}
	})

	t.Run("recurses into directories", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		dst := filepath.Join(dir, "dst")
		if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "sub", "a.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := Copy(src, dst); err != nil {
			t.Fatalf("Copy: %v", err)
		}
		if got, _ := os.ReadFile(filepath.Join(dst, "sub", "a.txt")); string(got) != "x" {
			t.Errorf("nested file = %q, want %q", got, "x")
		}
	})

	t.Run("recreates a symlink without following it", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "link")
		dst := filepath.Join(dir, "dst")
		if err := os.Symlink("t.txt", src); err != nil { // relative target, kept verbatim
			t.Fatal(err)
		}

		if _, err := Copy(src, dst); err != nil {
			t.Fatalf("Copy: %v", err)
		}
		info, err := os.Lstat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatal("dst is not a symlink")
		}
		if target, _ := os.Readlink(dst); target != "t.txt" {
			t.Errorf("symlink target = %q, want %q", target, "t.txt")
		}
	})
}
