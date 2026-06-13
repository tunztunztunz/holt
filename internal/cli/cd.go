package cli

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"charm.land/huh/v2"
	"github.com/sahilm/fuzzy"
	"github.com/tunztunztunz/acre/internal/state"
)

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
		return pick(query, matches)
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

// pick presents an interactive selector over ambiguous fuzzy matches. It uses
// huh, rendered on stderr (and reading stdin), so stdout stays the clean path
// the shell function captures — the same contract the old hand-rolled prompt
// kept, now reusable for any future selector.
func pick(query string, matches []*state.Record) (*state.Record, error) {
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
		Title(fmt.Sprintf("%d worktrees match %q", len(matches), query)).
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
