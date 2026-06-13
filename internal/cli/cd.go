package cli

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
	"github.com/tunztunztunz/acre/internal/state"
)

var (
	pickTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	pickIndex  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pickName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	pickBranch = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	pickPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
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

func pick(query string, matches []*state.Record) (*state.Record, error) {
	fmt.Fprintln(os.Stderr, pickTitle.Render(
		fmt.Sprintf("%d worktrees match %q:", len(matches), query)))

	for i, r := range matches {
		fmt.Fprintln(os.Stderr, lipgloss.JoinHorizontal(
			lipgloss.Left,
			pickIndex.Render(fmt.Sprintf(" %2d  ", i+1)),
			pickName.Render(r.SiteName),
			"  ",
			pickBranch.Render(r.Branch),
		))
	}
	fmt.Fprint(os.Stderr, pickPrompt.Render("› select: "))

	var choice int
	if _, err := fmt.Fscanln(os.Stdin, &choice); err != nil || choice < 1 || choice > len(matches) {
		return nil, Exitf(ExitUsage, "invalid selection")
	}
	return matches[choice-1], nil
}
