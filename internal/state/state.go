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
	return filepath.Join(repoRoot, ".acre")
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
