package vars

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tunztunztunz/acre/internal/config"
)

type Vars map[string]string

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func Resolve(repoRoot, branch string, p *config.Profile) (Vars, error) {
	v := Vars{
		"REPO_ROOT": repoRoot,
		"PROJECT":   filepath.Base(repoRoot),
		"BRANCH":    branch,
		"TREE":      strings.ReplaceAll(branch, "/", "-"),
	}
	site, err := v.expand(p.SiteName)
	if err != nil {
		return nil, err
	}
	if !isValidSlug(site) {
		return nil, fmt.Errorf("site_name %q expands to an invalid slug (template %q)", site, p.SiteName)
	}

	v["SITE_NAME"] = site
	v["WORKTREE"] = filepath.Join(repoRoot, p.WorktreesDir, site)

	return v, nil
}

func (v Vars) expand(tmpl string) (string, error) {
	var missing string
	out := os.Expand(tmpl, func(key string) string {
		if val, ok := v[key]; ok {
			return val
		}
		missing = key
		return ""
	})
	if missing != "" {
		return "", fmt.Errorf("undefined variable $%s in %q", missing, tmpl)
	}

	return out, nil
}
func isValidSlug(s string) bool {
	return slugRegex.MatchString(s)
}
