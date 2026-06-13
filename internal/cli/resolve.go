package cli

import (
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

func provideStore() (*state.Store, error) {
	root, err := provideRoot()
	if err != nil {
		return nil, err
	}
	s, err := state.Load(string(root))
	if err != nil {
		return nil, Exitf(ExitUsage, "%v", err)
	}
	return s, nil
}
