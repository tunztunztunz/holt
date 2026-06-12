package cli

import (
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tunztunztunz/acre/internal/config"
	"github.com/tunztunztunz/acre/internal/gitx"
	"github.com/tunztunztunz/acre/internal/provision"
	"github.com/tunztunztunz/acre/internal/state"
	"github.com/tunztunztunz/acre/internal/vars"
)

func newNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <branch>",
		Short: "Create and provision a worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]
			root, err := gitx.RepoRoot()
			if err != nil {
				return Exitf(ExitUsage, "%v", err)
			}

			profile, err := config.Load(root)
			if err != nil {
				return Exitf(ExitUsage, "%v", err)
			}

			if err := profile.Validate(); err != nil {
				return Exitf(ExitUsage, "%v", err)
			}

			v, err := vars.Resolve(root, branch, profile)
			if err != nil {
				return Exitf(ExitUsage, "%v", err)
			}

			store, err := state.Load(root)
			if err != nil {
				return Exitf(ExitUsage, "%v", err)
			}

			if _, ok := store.Worktrees[v.SiteName]; ok {
				return Exitf(ExitConflict, "worktree %s already exists", v.SiteName)
			}

			taken := make(map[int]bool)
			for _, t := range store.Worktrees {
				taken[t.Port] = true
			}

			port, err := provision.AllocatePort(profile.Port, v.SiteName, taken)
			if err != nil {
				return Exitf(ExitUsage, "%v", err)
			}
			v.Port = port

			if err := gitx.WorktreeAdd(root, v.Worktree, branch, true); err != nil {
				return Exitf(ExitConflict, "%v", err)
			}

			rec := &state.Record{
				SiteName:     v.SiteName,
				Branch:       branch,
				Path:         v.Worktree,
				Port:         port,
				Status:       "provisioning",
				CreatedAt:    time.Now(),
				LastActivity: time.Now(),
			}
			store.Worktrees[v.SiteName] = rec

			if err = state.Save(root, store); err != nil {
				return Exitf(ExitRuntime, "%v", err)
			}

			if err := provisionWorktree(root, v, profile); err != nil {
				rec.Status = "broken"
				_ = state.Save(root, store)
				return Exitf(ExitRuntime, "%v", err)
			}

			rec.Status = "ready"
			rec.LastActivity = time.Now()
			if err := state.Save(root, store); err != nil {
				return Exitf(ExitRuntime, "%v", err)
			}

			resultf("%s\n", v.Worktree)
			return nil
		},
	}
}

// provisionWorktree runs the copy, env, and setup steps against an already
// created worktree. It returns the first error encountered, leaving the
// caller to decide how to record the failure.
func provisionWorktree(root string, v *vars.Vars, p *config.Profile) error {
	for _, entry := range p.Copy {
		e, err := v.Expand(entry)
		if err != nil {
			return err
		}

		skip, err := provision.Copy(filepath.Join(root, e), filepath.Join(v.Worktree, e))
		if err != nil {
			return err
		}
		if skip != "" {
			warnf("copy %s: %s", e, skip)
		}
	}

	for _, block := range p.Env {
		if err := provision.RenderEnv(v.Worktree, block, v.Expand); err != nil {
			return err
		}
	}

	env := v.Environ()
	for _, line := range p.Setup {
		l, err := v.Expand(line)
		if err != nil {
			return err
		}
		if err := provision.RunCommand(v.Worktree, l, env); err != nil {
			return err
		}
	}

	return nil
}
