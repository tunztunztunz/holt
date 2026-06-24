package cli

import (
	"cmp"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
)

// pickBranch returns the branch to use: the arg if given, else an interactive
// list (TTY only) of local branches not already checked out in a worktree.
// Picking one attaches it; the "＋ new branch…" entry prompts for a name to
// create.
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
	// Nothing to attach so go straight to naming a new branch.
	if len(avail) == 0 {
		return promptNewBranch()
	}

	// Show the available branches as a (filterable) list, plus a sentinel entry
	// that routes to free-text entry for a brand-new branch. "" can't be a real
	// branch name, so it's a safe sentinel value.
	opts := make([]huh.Option[string], 0, len(avail)+1)
	for _, b := range avail {
		opts = append(opts, huh.NewOption(b, b))
	}
	opts = append(opts, huh.NewOption("＋ new branch…", ""))

	var chosen string
	sel := huh.NewSelect[string]().
		Title("Select a branch to attach, or create a new one").
		Options(opts...).
		Height(min(len(opts)+2, 12)).
		Value(&chosen)

	form := huh.NewForm(huh.NewGroup(sel)).
		WithOutput(os.Stderr).
		WithInput(os.Stdin)
	if err = form.Run(); err != nil {
		return "", Exitf(ExitUsage, "selection cancelled")
	}
	if chosen != "" {
		return chosen, nil // an existing branch
	}
	return promptNewBranch() // sentinel chosen → name a new branch
}

// promptNewBranch asks for a brand-new branch name on stderr.
func promptNewBranch() (string, error) {
	var name string
	in := huh.NewInput().
		Title("New branch name").
		Value(&name)

	form := huh.NewForm(huh.NewGroup(in)).
		WithOutput(os.Stderr).
		WithInput(os.Stdin)
	if err := form.Run(); err != nil {
		return "", Exitf(ExitUsage, "selection cancelled")
	}
	if name = strings.TrimSpace(name); name == "" {
		return "", Exitf(ExitNotFound, "no branch given")
	}
	return name, nil
}

// availableBranches lists local branches not already checked out in a worktree
// (git forbids the same branch in two worktrees), so they can be attached.
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

// baseFor resolves a base branch: the recorded fork point, else the configured
// base, else the repo default branch. The default is only queried when the
// first two are empty.
func baseFor(recorded string, profile *config.Profile) string {
	if b := cmp.Or(recorded, profile.Base); b != "" {
		return b
	}
	return gitx.DefaultBranch()
}
