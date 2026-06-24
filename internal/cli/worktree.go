package cli

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/sahilm/fuzzy"
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/provision"
	"github.com/tunztunztunz/holt/internal/state"
	"github.com/tunztunztunz/holt/internal/vars"
)

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

// pickMulti presents an interactive multi-selector over worktree records,
// returning them in selection order. Mirrors pick: caller builds matches and
// supplies title. Rendered on stderr (reading stdin) so stdout stays clean.
func pickMulti(title string, matches []*state.Record) ([]*state.Record, error) {
	opts := make([]huh.Option[*state.Record], len(matches))
	for i, r := range matches {
		label := r.SiteName
		if r.Branch != "" {
			label += "  " + r.Branch
		}
		opts[i] = huh.NewOption(label, r)
	}

	var chosen []*state.Record
	ms := huh.NewMultiSelect[*state.Record]().
		Title(title).
		Options(opts...).
		Height(min(len(opts)+2, 12)). // fit the list; cap so long lists still scroll
		Value(&chosen)

	form := huh.NewForm(huh.NewGroup(ms)).
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
		slices.SortFunc(all, func(a, b *state.Record) int {
			// Pin the main worktree to the top (cd injects it as "main"); order
			// the rest most-recent-first, like ls. rm's store has no main record,
			// so it just gets the recency sort.
			switch {
			case a.Status == "main":
				return -1
			case b.Status == "main":
				return 1
			default:
				return b.LastActivity.Compare(a.LastActivity)
			}
		})
		return pick(prompt, all)
	default:
		return resolveWorktree(arg, store.Worktrees)
	}
}

// pickOrdered resolves the selected worktrees, preserving integration order.
// Named args are resolved in the order given. With no args, it falls back to an
// interactive multi-select (requires a TTY on stderr); the selection order is
// the order returned. Refuses rather than guess when there's nothing to pick or
// no terminal to prompt on.
func pickOrdered(args []string, store *state.Store) ([]*state.Record, error) {
	if len(args) > 0 {
		out := make([]*state.Record, 0, len(args))
		for _, a := range args {
			rec, err := resolveWorktree(a, store.Worktrees)
			if err != nil {
				return nil, err
			}
			out = append(out, rec)
		}
		return out, nil
	}

	if !isTTY(os.Stderr) {
		return nil, Exitf(ExitUsage, "no worktrees given (and not a terminal)")
	}

	all := slices.Collect(maps.Values(store.Worktrees))
	if len(all) == 0 {
		return nil, Exitf(ExitNotFound, "no worktrees")
	}
	slices.SortFunc(all, func(a, b *state.Record) int {
		return a.LastActivity.Compare(b.LastActivity)
	})

	chosen, err := pickMulti("Select worktrees to harvest", all)
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, Exitf(ExitNotFound, "no worktrees selected")
	}
	return chosen, nil
}

// prepareWorktree resolves the vars for a new branch's worktree and refuses if
// one with that site name already exists.
func prepareWorktree(repo, branch string, profile *config.Profile, store *state.Store) (*vars.Vars, error) {
	v, err := vars.Resolve(repo, branch, profile)
	if err != nil {
		return nil, Exitf(ExitUsage, "%v", err)
	}
	if _, ok := store.Worktrees[v.SiteName]; ok {
		return nil, Exitf(ExitConflict, "worktree %q already exists", v.SiteName)
	}
	return v, nil
}

// resolveWorktree turns a user query into a worktree record. Worktrees are
// identified by NAME (site_name), never by branch — a tree can sit in detached
// HEAD with no branch at all, so a branch isn't a reliable handle. It matches an exact
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

// finalizeWorktree provisions the tree (when runProvision is set), flips the
// record to its terminal status, and persists state.
func finalizeWorktree(repo string, v *vars.Vars, profile *config.Profile, rec *state.Record, store *state.Store, runProvision bool) error {
	if runProvision {
		if err := provisionWorktree(repo, v, profile); err != nil {
			rec.Status = "broken"
			_ = state.Save(repo, store)
			return Exitf(ExitRuntime, "%v", err)
		}
	}
	rec.Status = "ready"
	rec.LastActivity = time.Now()
	if err := state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}
	return nil
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

// provisioningRecord builds the initial "provisioning" record for a tree.
func provisioningRecord(v *vars.Vars, branch, base string) *state.Record {
	return &state.Record{
		SiteName:     v.SiteName,
		Branch:       branch,
		BaseBranch:   base,
		Path:         v.Worktree,
		Port:         v.Port, // 0 when not provisioning
		Status:       "provisioning",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
}

// allocatePort picks a free port for siteName given what's already in use.
func allocatePort(profile *config.Profile, siteName string, store *state.Store) (int, error) {
	taken := make(map[int]bool)
	for _, t := range store.Worktrees {
		taken[t.Port] = true
	}
	port, err := provision.AllocatePort(profile.Port, siteName, taken)
	if err != nil {
		return 0, Exitf(ExitUsage, "%v", err)
	}
	return port, nil
}

// isInside reports whether the process cwd is within path.
func isInside(path string) bool {
	cwd, err := os.Getwd()
	return err == nil && strings.HasPrefix(cwd+"/", path+"/")
}
