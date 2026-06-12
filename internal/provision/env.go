package provision

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tunztunztunz/acre/internal/config"
)

func RenderEnv(worktree string, b config.EnvBlock, expand func(string) (string, error)) error {
	file := b.File
	if file == "" {
		file = ".env"
	}
	path := filepath.Join(worktree, file)

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var lines []string
	if len(content) > 0 {
		lines = strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	}

	remaining := map[string]bool{}
	for k := range b.Set {
		remaining[k] = true
	}

	for i, line := range lines {
		if k, ok := keyOf(line); ok {
			if raw, want := b.Set[k]; want {
				val, err := expand(raw)
				if err != nil {
					return err
				}
				lines[i] = k + "=" + val
				delete(remaining, k)
			}
		}
	}

	for _, k := range sortedKeys(remaining) {
		val, err := expand(b.Set[k])
		if err != nil {
			return err
		}
		lines = append(lines, k+"="+val)
	}

	present := map[string]bool{}
	for _, line := range lines {
		if k, ok := keyOf(line); ok {
			present[k] = true
		}
	}

	absent := map[string]bool{}
	for k := range b.Ensure {
		if !present[k] {
			absent[k] = true
		}
	}

	for _, k := range sortedKeys(absent) {
		val, err := expand(b.Ensure[k])
		if err != nil {
			return err
		}
		lines = append(lines, k+"="+val)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// sortedKeys returns a map's keys in stable alphabetical order, so appended
// lines don't shuffle run-to-run (Go map iteration is randomized).
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func keyOf(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", false // blank or comment, not an assignment
	}
	k, _, found := strings.Cut(t, "=")
	if !found {
		return "", false
	}
	return strings.TrimSpace(k), true
}
