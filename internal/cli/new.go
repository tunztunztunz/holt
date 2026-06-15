package cli

import (
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
)

// newCmd creates a worktree for a branch: it attaches an existing local branch
// or creates a new one off HEAD, allocates a free port, provisions the tree, and
// prints the new path so the holt() shell wrapper cds into it. With no branch
// arg it prompts, suggesting local branches not already in a worktree.
type newCmd struct {
	Branch string `arg:"" optional:"" help:"Branch to attach (if it exists) or create. Omit to pick."`
}

func (c *newCmd) Run(root Root, profile *config.Profile, store *state.Store) error {
	repo := string(root)

	if err := profile.Validate(); err != nil {
		return Exitf(ExitUsage, "%v", err)
	}

	branch, err := pickBranch(c.Branch, repo)
	if err != nil {
		return err
	}
	newBranch := !gitx.BranchExists(repo, branch)

	v, err := prepareWorktree(repo, branch, profile, store)
	if err != nil {
		return err
	}
	if v.Port, err = allocatePort(profile, v.SiteName, store); err != nil {
		return err
	}

	// Attach the existing branch, or create it off the current HEAD.
	if err = gitx.WorktreeAdd(repo, v.Worktree, branch, newBranch); err != nil {
		return Exitf(ExitConflict, "%v", err)
	}

	rec := provisioningRecord(v, branch, baseForNew(repo, newBranch, profile))
	store.Worktrees[v.SiteName] = rec
	if err = state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}

	if err = finalizeWorktree(repo, v, profile, rec, store, true); err != nil {
		return err
	}

	resultf("%s\n", v.Worktree)
	return nil
}

// baseForNew records where the worktree forks from: a freshly created branch
// forks off the current HEAD; an attached existing branch has no obvious fork
// point, so fall back to the configured base, else the repo default.
func baseForNew(repo string, newBranch bool, profile *config.Profile) string {
	if newBranch {
		return gitx.CurrentBranch(repo)
	}
	if profile.Base != "" {
		return profile.Base
	}
	return gitx.DefaultBranch()
}

// pickBranch returns the branch to use: the arg if given, else an interactive
// prompt (TTY only) suggesting local branches not already checked out in a
// worktree. Typed input matching a branch attaches it; anything else creates.
func pickBranch(arg, repo string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if !isTTY(os.Stderr) {
		return "", Exitf(ExitUsage, "branch required (nothing given, and not a terminal)")
	}

	avail, err := availableBranches(repo)
	if err != nil {
		return "", Exitf(ExitRuntime, "%v", err)
	}

	var branch string
	in := huh.NewInput().
		Title("Branch to attach or create").
		Description("an existing branch attaches it; a new name creates it").
		Suggestions(avail).
		Value(&branch)

	form := huh.NewForm(huh.NewGroup(in)).
		WithOutput(os.Stderr).
		WithInput(os.Stdin)
	if err = form.Run(); err != nil {
		return "", Exitf(ExitUsage, "selection cancelled")
	}
	if branch = strings.TrimSpace(branch); branch == "" {
		return "", Exitf(ExitNotFound, "no branch given")
	}
	return branch, nil
}

// availableBranches lists local branches not already checked out in a worktree,
// so they can be attached.
func availableBranches(repo string) ([]string, error) {
	all, err := gitx.LocalBranches(repo)
	if err != nil {
		return nil, err
	}
	live, err := gitx.WorktreeList(repo)
	if err != nil {
		return nil, err
	}

	taken := make(map[string]bool, len(live))
	for _, w := range live {
		if w.Branch != "" {
			taken[w.Branch] = true
		}
	}

	out := make([]string, 0, len(all))
	for _, b := range all {
		if !taken[b] {
			out = append(out, b)
		}
	}
	return out, nil
}
