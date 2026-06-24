package cli

import (
	"path/filepath"

	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
)

// CdCmd resolves a worktree and prints its path to stdout; it never changes
// directory itself — the holt() shell wrapper cds into whatever path it prints.
// The main worktree (untracked in state) is injected as a target for this
// invocation only.
type CdCmd struct {
	Name string `arg:"" optional:"" help:"Worktree to move to. Omit to pick from all; 'here' is a no-op."`
}

func (c *CdCmd) Run(root Root, store *state.Store) error {
	// The main worktree isn't tracked in state (reconcile skips the repo root),
	// but it's a valid cd target. Inject it into this invocation's picker only.
	// cd never saves, and rm gets a fresh store, so this can't leak into removals.
	addMain(root, store)

	rec, err := pickTarget(c.Name, store, "Select a worktree to move to")
	if err != nil {
		return err
	}
	if isInside(rec.Path) {
		return Exitf(ExitUsage, "You are already inside the selected worktree")
	}

	resultf("%s\n", rec.Path)
	return nil
}

// addMain adds a synthetic record for the main worktree (repo root) so cd can
// navigate back to it by name, fuzzy match, or from the no-arg picker.
func addMain(root Root, store *state.Store) {
	repo := string(root)
	name := filepath.Base(repo)

	branch := ""
	if live, err := gitx.WorktreeList(repo); err == nil {
		for _, w := range live {
			if w.Path == repo {
				branch = w.Branch
				break
			}
		}
	}

	store.Worktrees[name] = &state.Record{
		SiteName: name,
		Branch:   branch,
		Path:     repo,
		Status:   "main",
	}
}
