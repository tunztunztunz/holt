package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func Load(repoRoot string) (*Profile, error) {
	path := filepath.Join(repoRoot, "acre.yml")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no acre.yml in %s (run: acre init)", repoRoot)
	}
	defer func() { _ = f.Close() }()

	var p Profile
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("invalid acre.yml: %w", err)
	}
	p.applyDefaults()
	return &p, nil
}

func (p *Profile) applyDefaults() {
	if p.Version == 0 {
		p.Version = 1
	}
	if p.SiteName == "" {
		p.SiteName = "$PROJECT-$TREE"
	}
	if p.WorktreesDir == "" {
		p.WorktreesDir = ".." // sibling layout by default, matching `acre init`
	}
	if len(p.Guards) == 0 {
		p.Guards = []string{"uncommitted", "unmerged"}
	}
}
