package vars

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tunztunztunz/acre/internal/config"
)

var slugRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Vars holds the substitution variables for a worktree.
type Vars struct {
	RepoRoot string
	Project  string
	Branch   string
	Tree     string
	SiteName string
	Worktree string
	Port     int
}

// kv is a single variable name/value pair.
type kv struct{ name, value string }

// Resolve builds the variable set for branch under repoRoot.
func Resolve(repoRoot, branch string, p *config.Profile) (*Vars, error) {
	v := &Vars{
		RepoRoot: repoRoot,
		Project:  filepath.Base(repoRoot),
		Branch:   branch,
		Tree:     strings.ReplaceAll(branch, "/", "-"),
	}

	site, err := v.Expand(p.SiteName)
	if err != nil {
		return nil, err
	}
	if !isValidSlug(site) {
		return nil, fmt.Errorf("site_name %q expands to an invalid slug", site)
	}

	v.SiteName = site
	v.Worktree = worktreePath(repoRoot, p.WorktreesDir, site)

	return v, nil
}

// pairs is the single source of truth for variable names and their values.
// Expand, Environ, and lookup all derive from it, so a new variable only
// needs to be added here (and to the struct).
func (v *Vars) pairs() []kv {
	return []kv{
		{"REPO_ROOT", v.RepoRoot},
		{"PROJECT", v.Project},
		{"BRANCH", v.Branch},
		{"TREE", v.Tree},
		{"SITE_NAME", v.SiteName},
		{"WORKTREE", v.Worktree},
		{"PORT", strconv.Itoa(v.Port)},
	}
}

// Expand substitutes $VAR / ${VAR} references in tmpl, returning an error
// if any referenced variable is undefined.
func (v *Vars) Expand(tmpl string) (string, error) {
	var missing string

	out := os.Expand(tmpl, func(key string) string {
		val, ok := v.lookup(key)
		if !ok {
			missing = key
		}
		return val
	})

	if missing != "" {
		return "", fmt.Errorf("undefined variable $%s in %q", missing, tmpl)
	}

	return out, nil
}

// Environ renders the variables as "KEY=value" strings for a subprocess environment.
func (v *Vars) Environ() []string {
	pairs := v.pairs()
	env := make([]string, len(pairs))
	for i, p := range pairs {
		env[i] = p.name + "=" + p.value
	}
	return env
}

func (v *Vars) lookup(name string) (string, bool) {
	for _, p := range v.pairs() {
		if p.name == name {
			return p.value, true
		}
	}
	return "", false
}

func isValidSlug(s string) bool { return slugRe.MatchString(s) }

func worktreePath(repoRoot, worktreesDir, site string) string {
	base := worktreesDir
	if strings.HasPrefix(base, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, base[2:])
		}
	}

	if !filepath.IsAbs(base) {
		base = filepath.Join(repoRoot, base)
	}

	return filepath.Join(base, site)
}
