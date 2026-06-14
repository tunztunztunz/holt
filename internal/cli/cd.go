package cli

import "github.com/tunztunztunz/acre/internal/state"

type CdCmd struct {
	Name string `arg:"" help:"Worktree name to resolve to a path."`
}

func (c *CdCmd) Run(store *state.Store) error {
	rec, err := resolveWorktree(c.Name, store.Worktrees)
	if err != nil {
		return err
	}
	resultf("%s\n", rec.Path)
	return nil
}
