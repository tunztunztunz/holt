package cli

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
	"github.com/tunztunztunz/holt/internal/ui"
	"github.com/tunztunztunz/holt/internal/vars"
)

type harvestPlan struct {
	base     string
	clean    []*state.Record // verified clean, in integration order
	conflict *conflictInfo   // first conflict, or nil if all clean
}

type conflictInfo struct {
	rec   *state.Record
	files []string
}

// HarvestCmd forecasts integrating a set of worktrees onto their shared base
// branch. It resolves the base from the selection (unless --base overrides),
// then probes the merges cumulatively and reports how far they go clean.
type HarvestCmd struct {
	Names []string `arg:"" optional:"" help:"Worktrees to integrate, in order. Omit to pick interactively."`
	Base  string   `help:"Branch to integrate onto. Overrides the trees' recorded base." name:"base"`
	Into  string   `help:"Name for the new integration branch (required to execute)." name:"into"`

	Provision bool `help:"Provision the integration tree so it's runnable." negatable:"" default:"true"`
}

func (c *HarvestCmd) Run(root Root, profile *config.Profile, store *state.Store, g *Globals) error {
	repo := string(root)

	recs, err := pickOrdered(c.Names, store)
	if err != nil {
		return err
	}

	if c.Into == "" {
		return Exitf(ExitUsage, "name the integration branch with --into <branch>")
	}
	intBranch := c.Into

	v, err := vars.Resolve(repo, intBranch, profile)
	if err != nil {
		return Exitf(ExitUsage, "%v", err)
	}
	rec, exists := store.Worktrees[v.SiteName]

	// What we forecast and merge against: an existing integration tree takes the
	// new trees onto its own HEAD; a fresh one is created off the selection's base.
	base := intBranch
	if !exists {
		if base, err = baseForSelection(recs, c.Base); err != nil {
			return err
		}
	}

	plan, err := planHarvest(repo, base, recs)
	if err != nil {
		return err
	}
	renderPlan(plan)
	if g.DryRun {
		return nil
	}

	prompt := fmt.Sprintf("merge %d tree(s) into %s?", len(recs), intBranch)
	if !exists {
		prompt = fmt.Sprintf("create %s and merge %d tree(s)?", intBranch, len(recs))
	}
	if !g.Force && !confirm(prompt) {
		return Exitf(ExitOK, "")
	}

	// Create the integration tree on the first run; reuse it on later ones.
	if !exists {
		if c.Provision {
			if err = profile.Validate(); err != nil {
				return Exitf(ExitUsage, "%v", err)
			}
			if v.Port, err = allocatePort(profile, v.SiteName, store); err != nil {
				return err
			}
		}
		// base is only a start-point — not checked out, so a live base is fine.
		if err = gitx.WorktreeAddFrom(repo, v.Worktree, intBranch, base); err != nil {
			return Exitf(ExitConflict, "%v", err)
		}
		rec = provisioningRecord(v, intBranch, base)
		store.Worktrees[v.SiteName] = rec
		if err = state.Save(repo, store); err != nil {
			return Exitf(ExitRuntime, "%v", err)
		}
	}

	// Merge in order. First conflict is a STOP, not a cleanup: the clean prefix is
	// already committed in intBranch. Resolve in the tree and commit; if trees
	// remain, re-run with them — harvest merges them into this same tree.
	for i, r := range recs {
		if err = gitx.Merge(v.Worktree, r.Branch); err != nil {
			if rest := recs[i+1:]; len(rest) > 0 {
				names := make([]string, len(rest))
				for j, rr := range rest {
					// Short, copy-pasteable handle: drop the project prefix
					// ($PROJECT-$TREE -> $TREE); resolveWorktree matches it back.
					names[j] = strings.TrimPrefix(rr.SiteName, v.Project+"-")
				}
				warnf("conflict merging %s — resolve in %s, commit, then re-run with the rest:", r.SiteName, v.Worktree)
				warnf("  holt harvest %s --into %s", strings.Join(names, " "), intBranch)
			} else {
				warnf("conflict merging %s — resolve in %s and commit to finish", r.SiteName, v.Worktree)
			}
			return Exitf(ExitConflict, "stopped at first conflict (%s)", r.SiteName)
		}
		infof("  ✓ merged %s", r.SiteName)
	}

	// Provision only when we created the tree; an existing one is already set up.
	if err = finalizeWorktree(repo, v, profile, rec, store, c.Provision && !exists); err != nil {
		return err
	}

	resultf("%s\n", v.Worktree) // stdout = integration path; cd "$(holt harvest …)"
	return nil
}

// baseForSelection resolves the branch to integrate onto. --base always wins;
// otherwise the recorded fork point (Record.BaseBranch) decides, but only if the
// whole selection agrees. Divergent or missing bases refuse rather than guess.
func baseForSelection(recs []*state.Record, override string) (string, error) {
	if override != "" {
		return override, nil // --base always wins
	}

	seen := map[string]bool{}
	for _, r := range recs {
		if r.BaseBranch != "" {
			seen[r.BaseBranch] = true
		}
	}

	branches := slices.Sorted(maps.Keys(seen))
	switch len(branches) {
	case 1:
		return branches[0], nil
	case 0:
		return "", Exitf(ExitNotFound, "none of the selected worktrees have a recorded base branch; pass --base <branch>")
	default:
		return "", renderDivergentBase(recs)
	}
}

// planHarvest probes the integration cumulatively: each branch is merged (in
// memory) against the accumulated result of the clean ones before it, not just
// against base, so "the first N are clean" is actually true. It stops at the
// first conflict on purpose. Conflicts past that point depend on how the user
// resolves this one, so they aren't knowable and aren't forecast.
func planHarvest(repo, base string, order []*state.Record) (*harvestPlan, error) {
	plan := &harvestPlan{base: base}
	acc := base // accumulated commit-ish, starts at the base branch

	for _, rec := range order {
		tree, conflicts, err := gitx.MergeTree(repo, acc, rec.Branch)
		if err != nil {
			return nil, Exitf(ExitRuntime, "%v", err)
		}
		if len(conflicts) > 0 {
			plan.conflict = &conflictInfo{rec: rec, files: conflicts}
			break // forecast stops at the first conflict
		}
		plan.clean = append(plan.clean, rec)

		acc, err = gitx.CommitTree(repo, tree, acc) // chain: next probe is vs this result
		if err != nil {
			return nil, Exitf(ExitRuntime, "%v", err)
		}
	}

	return plan, nil
}

func renderPlan(plan *harvestPlan) {
	if len(plan.clean) == 0 && plan.conflict == nil {
		infof("nothing to harvest")
		return
	}

	infof("integrating onto %s:", plan.base)
	for _, r := range plan.clean {
		infof("  ✓ %s (%s) — clean", r.SiteName, r.Branch)
	}

	if plan.conflict != nil {
		c := plan.conflict
		warnf("  ✗ %s (%s) — conflicts in %d file(s): %s",
			c.rec.SiteName, c.rec.Branch, len(c.files), strings.Join(c.files, ", "))
		warnf("forecast stops here — later worktrees depend on how you resolve this")
	}
}

// renderDivergentBase prints a tree/branch/base table to stderr and refuses,
// leaving it to the user to disambiguate with --base.
func renderDivergentBase(recs []*state.Record) error {
	t := ui.Table("TREE", "BRANCH", "BASE")
	for _, r := range recs {
		t.Row(r.SiteName, r.Branch, cmp.Or(r.BaseBranch, "(none)"))
	}
	fmt.Fprintln(os.Stderr, t)
	return Exitf(ExitUsage, "selected worktrees have divergent base branches; pass --base <branch>")
}
