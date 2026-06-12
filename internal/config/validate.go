package config

import (
	"fmt"
	"path/filepath"
)

var knownGuards = map[string]bool{
	"uncommitted": true,
	"unmerged":    true,
	"unpushed":    true,
	"stashes":     true,
}

func (p *Profile) Validate() error {
	if p.Port != nil {
		lo, hi := p.Port.Range[0], p.Port.Range[1]
		if lo < 1024 || hi > 65535 || lo > hi {
			return fmt.Errorf("port.range %v invalid (want ordered, 1024-65535)", p.Port.Range)
		}

		switch p.Port.Strategy {
		case "", PortHash, PortFree:
		default:
			return fmt.Errorf("unknown strategy: %q", p.Port.Strategy)
		}
	}

	for _, g := range p.Guards {
		if !knownGuards[g] {
			return fmt.Errorf("unknown guard %q", g)
		}
	}

	for _, c := range p.Copy {
		if err := validateRelPath("copy", c); err != nil {
			return err
		}
	}
	for _, e := range p.Env {
		if err := validateRelPath("env.file", e.File); err != nil {
			return err
		}
	}
	for _, l := range p.Link {
		if err := validateRelPath("link", l); err != nil {
			return err
		}
	}

	return nil
}

func validateRelPath(kind, p string) error {
	if !filepath.IsLocal(p) {
		return fmt.Errorf("%s path must stay inside the worktree: %q", kind, p)
	}
	return nil
}
