package provision

import (
	"io"
	"os"
	"path/filepath"
)

// Copy copies src to dst, recursing into directories, preserving symlinks and
// mode bits. It returns a non-empty `skipped` reason (with nil error) for the
// documented no-op cases so the caller can warn instead of failing.
func Copy(src, dst string) (skipped string, err error) {
	// Lstat, not Stat, so a symlink is seen as itself rather than its target.
	info, err := os.Lstat(src)
	if os.IsNotExist(err) {
		return "source missing", nil
	}
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(dst); err == nil {
		return "dest exists", nil
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, rerr := os.Readlink(src)
		if rerr != nil {
			return "", rerr
		}
		return "", os.Symlink(target, dst) // recreate the link, don't follow it

	case info.IsDir():
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return "", err
		}
		entries, rerr := os.ReadDir(src)
		if rerr != nil {
			return "", rerr
		}
		for _, e := range entries {
			if _, cerr := Copy(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); cerr != nil {
				return "", cerr
			}
		}
		return "", nil

	default:
		return copyFile(src, dst, info)
	}
}

func copyFile(src, dst string, info os.FileInfo) (skipped string, err error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()

	// Create with the source's permission bits so e.g. a 0600 key stays 0600.
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return "", err
}
