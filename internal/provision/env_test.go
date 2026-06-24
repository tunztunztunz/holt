package provision

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tunztunztunz/holt/internal/config"
)

// identity is an expand func that returns values verbatim, so test inputs read
// as their literal expected output.
func identity(s string) (string, error) { return s, nil }

func TestKeyOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line    string
		wantKey string
		wantOK  bool
	}{
		{"", "", false},
		{"   ", "", false},
		{"# comment", "", false},
		{"FOO=bar", "FOO", true},
		{"  FOO = bar  ", "FOO", true}, // key is trimmed
		{"noequals", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			key, ok := keyOf(tt.line)
			if key != tt.wantKey || ok != tt.wantOK {
				t.Errorf("keyOf(%q) = (%q, %v), want (%q, %v)", tt.line, key, ok, tt.wantKey, tt.wantOK)
			}
		})
	}
}

func TestRenderEnv(t *testing.T) {
	t.Run("creates the file with Set then Ensure, sorted", func(t *testing.T) {
		dir := t.TempDir()
		b := config.EnvBlock{
			Set:    map[string]string{"PORT": "4000"},
			Ensure: map[string]string{"APP": "x"},
		}
		if err := RenderEnv(dir, b, identity); err != nil {
			t.Fatalf("RenderEnv: %v", err)
		}
		// Set keys appended first (sorted), then absent Ensure keys (sorted).
		assertEnv(t, dir, ".env", "PORT=4000\nAPP=x\n")
	})

	t.Run("replaces a Set key in place, preserving order and comments", func(t *testing.T) {
		dir := seedEnv(t, "# comment\nFOO=old\nBAR=keep\n")
		b := config.EnvBlock{Set: map[string]string{"FOO": "new"}}
		if err := RenderEnv(dir, b, identity); err != nil {
			t.Fatalf("RenderEnv: %v", err)
		}
		assertEnv(t, dir, ".env", "# comment\nFOO=new\nBAR=keep\n")
	})

	t.Run("appends a Set key that isn't present", func(t *testing.T) {
		dir := seedEnv(t, "FOO=1\n")
		b := config.EnvBlock{Set: map[string]string{"BAZ": "2"}}
		if err := RenderEnv(dir, b, identity); err != nil {
			t.Fatalf("RenderEnv: %v", err)
		}
		assertEnv(t, dir, ".env", "FOO=1\nBAZ=2\n")
	})

	t.Run("Ensure only adds keys that are absent", func(t *testing.T) {
		dir := seedEnv(t, "FOO=existing\n")
		b := config.EnvBlock{Ensure: map[string]string{"FOO": "default", "NEW": "n"}}
		if err := RenderEnv(dir, b, identity); err != nil {
			t.Fatalf("RenderEnv: %v", err)
		}
		assertEnv(t, dir, ".env", "FOO=existing\nNEW=n\n")
	})

	t.Run("expand errors propagate", func(t *testing.T) {
		dir := t.TempDir()
		b := config.EnvBlock{Set: map[string]string{"X": "$BAD"}}
		boom := func(string) (string, error) { return "", errors.New("boom") }
		if err := RenderEnv(dir, b, boom); err == nil {
			t.Fatal("want error from expand, got nil")
		}
	})

	t.Run("a freshly created dotenv is 0o600", func(t *testing.T) {
		dir := t.TempDir()
		b := config.EnvBlock{Set: map[string]string{"PORT": "4000"}}
		if err := RenderEnv(dir, b, identity); err != nil {
			t.Fatalf("RenderEnv: %v", err)
		}
		info, err := os.Stat(filepath.Join(dir, ".env"))
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("mode = %o, want 600", got)
		}
	})
}

// seedEnv writes an initial .env into a fresh temp dir and returns the dir.
func seedEnv(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}
	return dir
}

// assertEnv checks that dir/name has exactly the wanted contents.
func assertEnv(t *testing.T, dir, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if string(got) != want {
		t.Errorf("%s contents:\ngot:\n%q\nwant:\n%q", name, got, want)
	}
}
