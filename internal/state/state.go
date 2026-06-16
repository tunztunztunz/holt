package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	SiteName     string            `json:"site_name"`
	Branch       string            `json:"branch"`
	BaseBranch   string            `json:"base_branch,omitempty"`
	Path         string            `json:"path"`
	Status       string            `json:"status"`
	Port         int               `json:"port"`
	URL          string            `json:"url"`
	ProfileHash  string            `json:"profile_hash"`
	Resources    map[string]string `json:"resources"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"updated_at"`
}

type Store struct {
	Version   int                `json:"version"`
	UpdatedAt time.Time          `json:"updated_at"`
	Worktrees map[string]*Record `json:"worktrees"`
}

func dir(repoRoot string) string {
	return filepath.Join(repoRoot, ".holt")
}

func file(repoRoot string) string {
	return filepath.Join(dir(repoRoot), "state.json")
}

func Load(repoRoot string) (*Store, error) {
	b, err := os.ReadFile(file(repoRoot))
	if os.IsNotExist(err) {
		return &Store{Version: 1, Worktrees: map[string]*Record{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.Worktrees == nil {
		s.Worktrees = map[string]*Record{}
	}

	return &s, nil
}

// Save writes the store via a temp file + rename, so a reader never sees a
// half-written file. There's no lock: two concurrent holt runs do an
// unsynchronized read-modify-write, so the last to save wins and can drop the
// other's record. Acceptable for a single-user dev tool.
func Save(repoRoot string, s *Store) error {
	if err := os.MkdirAll(dir(repoRoot), 0o755); err != nil {
		return err
	}

	s.UpdatedAt = time.Now()

	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	tmp := file(repoRoot) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, file(repoRoot))
}
