package cli

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"github.com/sahilm/fuzzy"
	"github.com/tunztunztunz/acre/internal/state"
)

// resolveWorktree turns a user query into a worktree record. Worktrees are
// identified by NAME (site_name), never by branch — branches aren't unique to a
// tree (a detached HEAD, or several trees on one branch). It matches an exact
// name first, then — only when stderr is a TTY — fuzzy-matches and offers a
// picker; piped/CI callers must pass an exact name.
func resolveWorktree(query string, recs map[string]*state.Record) (*state.Record, error) {
	if rec, ok := recs[query]; ok {
		return rec, nil
	}

	if !isTTY(os.Stderr) {
		return nil, Exitf(ExitUsage, "no worktree named %q (provide an exact name)", query)
	}

	list := slices.Collect(maps.Values(recs))

	switch matches := fuzzyMatch(query, list); len(matches) {
	case 0:
		return nil, Exitf(ExitNotFound, "no worktree matches %q", query)
	case 1:
		return matches[0], nil
	default:
		return pick(fmt.Sprintf("%d worktrees match %q", len(matches), query), matches)
	}
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func fuzzyMatch(query string, recs []*state.Record) []*state.Record {
	names := make([]string, len(recs))
	for i, r := range recs {
		names[i] = r.SiteName
	}

	out := make([]*state.Record, 0, len(recs))
	for _, m := range fuzzy.Find(query, names) {
		out = append(out, recs[m.Index])
	}
	return out
}

// pick presents an interactive selector over worktree records. It uses huh,
// rendered on stderr (reading stdin), so stdout stays the clean path the shell
// function captures. title labels the prompt — callers pass context such as
// `3 worktrees match "fea"` or `Select a worktree to remove`.
func pick(title string, matches []*state.Record) (*state.Record, error) {
	opts := make([]huh.Option[*state.Record], len(matches))
	for i, r := range matches {
		label := r.SiteName
		if r.Branch != "" {
			label += "  " + r.Branch
		}
		opts[i] = huh.NewOption(label, r)
	}

	var chosen *state.Record
	sel := huh.NewSelect[*state.Record]().
		Title(title).
		Options(opts...).
		Value(&chosen)

	form := huh.NewForm(huh.NewGroup(sel)).
		WithOutput(os.Stderr).
		WithInput(os.Stdin)

	if err := form.Run(); err != nil {
		return nil, Exitf(ExitUsage, "selection cancelled")
	}
	return chosen, nil
}

// pickTarget resolves a command argument to ONE worktree: "here" → the tree the cwd is
// inside, "" → a picker over all trees, a name → resolveWorktree (exact → fuzzy → picker).
func pickTarget(arg string, store *state.Store, prompt string) (*state.Record, error) {
	switch arg {
	case "here":
		if rec := currentTree(store); rec != nil {
			return rec, nil
		}
		return nil, Exitf(ExitNotFound, "not inside a managed worktree")
	case "":
		if !isTTY(os.Stderr) {
			return nil, Exitf(ExitUsage, "name required (nothing given, and not a terminal)")
		}
		all := slices.Collect(maps.Values(store.Worktrees))
		if len(all) == 0 {
			return nil, Exitf(ExitNotFound, "no worktrees")
		}
		return pick(prompt, all)
	default:
		return resolveWorktree(arg, store.Worktrees)
	}
}

// currentTree returns the managed worktree the cwd is inside, or nil.
func currentTree(store *state.Store) *state.Record {
	for _, r := range store.Worktrees {
		if isInside(r.Path) {
			return r
		}
	}
	return nil
}

// isInside reports whether the process cwd is within path.
func isInside(path string) bool {
	cwd, err := os.Getwd()
	return err == nil && strings.HasPrefix(cwd+"/", path+"/")
}
