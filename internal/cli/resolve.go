package cli

import (
	"path/filepath"

	"github.com/tunztunztunz/acre/internal/config"
	"github.com/tunztunztunz/acre/internal/gitx"
	"github.com/tunztunztunz/acre/internal/state"
)

// Root is the repository root path. It's a named type so Kong can inject it
// into commands unambiguously (a bare string binding would be too broad).
type Root string

// The provide* functions are registered with Kong via BindToProvider and called
// lazily — only when a command's Run method declares that dependency. So
// version (no deps) never resolves a repo, and init (Root only) never loads a
// profile that doesn't exist yet.

func provideRoot() (Root, error) {
	root, err := gitx.RepoRoot()
	if err != nil {
		return "", Exitf(ExitUsage, "%v", err)
	}
	return Root(root), nil
}

// provideProfile is the single place acre reads acre.yml. It needs the root, so
// it just calls provideRoot itself.
func provideProfile() (*config.Profile, error) {
	root, err := provideRoot()
	if err != nil {
		return nil, err
	}
	p, err := config.Load(string(root))
	if err != nil {
		return nil, Exitf(ExitUsage, "%v", err)
	}
	return p, nil
}

// provideStore loads state AND reconciles it against git, so every command that
// declares *state.Store gets a git-true view for free.
func provideStore() (*state.Store, error) {
	root, err := provideRoot()
	if err != nil {
		return nil, err
	}
	sRoot := string(root)

	s, err := state.Load(sRoot)
	if err != nil {
		return nil, Exitf(ExitUsage, "%v", err)
	}

	live, err := gitx.WorktreeList(sRoot)
	if err != nil {
		return nil, Exitf(ExitUsage, "%v", err)
	}

	reconcile(s, live, sRoot)

	return s, nil
}

// reconcile makes the store agree with git. Git decides EXISTENCE; state only
// carries metadata. No disk write here — reads stay side-effect-free.
func reconcile(s *state.Store, live []gitx.Worktree, root string) {
	liveByPath := make(map[string]gitx.Worktree, len(live))
	for _, w := range live {
		liveByPath[w.Path] = w
	}

	managed := make(map[string]bool, len(s.Worktrees))
	for key, rec := range s.Worktrees {
		if _, ok := liveByPath[rec.Path]; !ok {
			delete(s.Worktrees, key)
			continue
		}
		managed[rec.Path] = true
	}

	for _, w := range live {
		if w.Path == root || managed[w.Path] {
			continue
		}
		name := filepath.Base(w.Path)
		s.Worktrees[name] = &state.Record{
			SiteName: name,
			Branch:   w.Branch,
			Path:     w.Path,
			Status:   "unmanaged",
		}
	}
}
