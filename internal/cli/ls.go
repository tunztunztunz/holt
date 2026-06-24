package cli

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
	"github.com/tunztunztunz/holt/internal/ui"
)

type lsRow struct {
	Name         string      `json:"name"`
	Branch       string      `json:"branch"`
	Port         int         `json:"port"`
	URL          string      `json:"url"`
	Status       string      `json:"status"`
	Git          gitx.Status `json:"git"`
	LastActivity time.Time   `json:"last_activity"`
}

// LsCmd lists worktrees with their resolved base branch and git status
// (dirty/ahead/behind), most-recently-active first. Output is a human table by
// default, --json for structured output, or --porcelain for stable
// tab-separated lines.
type LsCmd struct {
	Porcelain bool `help:"Stable tab-separated output for scripts."`
}

func (c *LsCmd) Run(profile *config.Profile, store *state.Store, g *Globals) error {
	rows := buildRows(store.Worktrees, profile)

	if g.JSON {
		return emitJSON("ls", rows)
	}
	if c.Porcelain {
		return emitPorcelain(rows)
	}
	return emitTable(rows)
}

// buildRows resolves each worktree's base branch (via baseFor) and reports its
// git status (ahead/behind) against it.
func buildRows(recs map[string]*state.Record, profile *config.Profile) []lsRow {
	rows := make([]lsRow, 0, len(recs))
	for _, r := range recs {
		gs, _ := gitx.WorktreeStatus(r.Path, baseFor(r.BaseBranch, profile))
		rows = append(rows, lsRow{
			Name:         r.SiteName,
			Branch:       r.Branch,
			Port:         r.Port,
			URL:          r.URL,
			Status:       r.Status,
			Git:          gs,
			LastActivity: r.LastActivity,
		})
	}

	slices.SortFunc(rows, func(a, b lsRow) int {
		return b.LastActivity.Compare(a.LastActivity)
	})

	return rows
}

// emitTable renders the human-facing table to stdout — this is the command result.
func emitTable(rows []lsRow) error {
	t := ui.Table("NAME", "BRANCH", "PORT", "GIT", "LAST", "STATUS")

	for _, r := range rows {
		t.Row(
			r.Name,
			r.Branch,
			portCell(r.Port),
			gitCell(r.Git),
			humanSince(r.LastActivity),
			r.Status,
		)
	}

	resultf("%s\n", t)
	return nil
}

// emitPorcelain writes one worktree per line, tab-separated, fixed field order,
// no color or padding — stable for scripts to parse with cut/awk.
// Fields: name branch port url status dirty ahead behind last_activity
func emitPorcelain(rows []lsRow) error {
	for _, r := range rows {
		port := ""
		if r.Port != 0 {
			port = strconv.Itoa(r.Port)
		}
		resultf("%s\t%s\t%s\t%s\t%s\t%t\t%d\t%d\t%s\n",
			r.Name,
			r.Branch,
			port,
			r.URL,
			r.Status,
			r.Git.Dirty,
			r.Git.Ahead,
			r.Git.Behind,
			r.LastActivity.UTC().Format(time.RFC3339),
		)
	}
	return nil
}

// gitCell renders status as ✓ (clean) or ● (dirty), with ↑ahead/↓behind appended.
func gitCell(s gitx.Status) string {
	glyph := "✓"
	if s.Dirty {
		glyph = "●"
	}
	track := ""
	if s.Ahead > 0 {
		track += fmt.Sprintf("↑%d", s.Ahead)
	}
	if s.Behind > 0 {
		track += fmt.Sprintf("↓%d", s.Behind)
	}
	if track != "" {
		return glyph + " " + track
	}
	return glyph
}

func humanSince(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	switch d := time.Since(t); {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func portCell(p int) string {
	if p == 0 {
		return "-"
	}
	return strconv.Itoa(p)
}
