package cli

import (
	"os"
	"path/filepath"
)

type initCmd struct{}

func (c *initCmd) Run(root Root, g *Globals) error {
	path := filepath.Join(string(root), "acre.yml")
	if _, err := os.Stat(path); err == nil && !g.Force {
		return Exitf(ExitConflict, "acre.yml already exists (use --force)")
	}
	if err := os.WriteFile(path, []byte(starterProfile), 0o644); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}
	infof("wrote %s", path)

	return nil
}

// starterProfile is the acre.yml that `acre init` scaffolds into a new repo.
// Stack-agnostic: universal bits active, framework-specific examples commented.
const starterProfile = `version: 1

# Identifier for each worktree and its site. Default: $PROJECT-$TREE
site_name: $PROJECT-$TREE

# Where worktrees are created. Resolves relative to the repo, or accepts an
# absolute path or ~ for home. Default ".." gives the recommended sibling
# layout next to the repo (../<repo>-<branch>), per git-worktree best practices.
#   ..                -> ../acre-feature-x   (sibling, recommended)
#   .acre/worktrees   -> inside the repo
#   ~/worktrees       -> one central place for every repo
worktrees_dir: ..

# Gitignored files to copy from the main worktree into each new one.
copy:
  - .env

# Large, branch-invariant dirs to symlink instead of copy (saves disk + time).
# link:
#   - node_modules
#   - vendor

# Rewrite per-worktree values into a dotenv file so trees don't collide.
# (Uncomment the port: block below if you reference $PORT here.)
# env:
#   - file: .env
#     set:
#       APP_URL: http://$SITE_NAME.test
#       PORT: $PORT

# Allocate a unique port per worktree, exposed as $PORT.
# port:
#   range: [4000, 4999]

# Commands run in each new worktree on 'acre new', in order.
setup: []

# Commands run in a worktree on 'acre rm', before removal.
teardown: []

# Warn + confirm before destructive actions (rm, gc).
guards:
  - uncommitted
  - unmerged
`
