# holt

A small CLI for working with git worktrees without the busywork.

Git worktrees let you check out several branches at once, each in its own
directory, all backed by a single clone. That's great for jumping between a
feature, a hotfix, and a review without stashing or rebuilding. The catch is
that every new worktree needs the same setup your main checkout has: copied env
files, installed dependencies, a fresh database, a local domain, and so on. And
when you're done, you have to tear all of that back down.

holt handles that part. You describe your project once in a `holt.yml`, and
then `holt new` spins up a fully provisioned worktree and drops you into it,
while `holt rm` cleans it up and walks you back home.

## How it works

Each repo gets a `holt.yml` that describes how to build a worktree: which files
to copy, which commands to run on setup and teardown, which branch your trees
fork from, and which safety checks to run before anything destructive. holt
keeps a little state file under `.holt/` so it can list your trees and reconcile
against what git actually has on disk.

Worktrees are identified by name (the site name), not by branch. Git only lets a
branch be checked out in one worktree at a time, and a tree can also sit in
detached HEAD with no branch at all, so the branch isn't a reliable handle. The
name always is.

## Install

You'll need Go 1.26 or newer.

From source:

```sh
git clone https://github.com/tunztunztunz/holt.git
cd holt
go install ./...
```

Or in one step:

```sh
go install github.com/tunztunztunz/holt@latest
```

That puts the `holt` binary in `$(go env GOPATH)/bin`, so make sure that's on
your `PATH`.

### Shell integration (required for new, cd, and rm)

A program can't change the working directory of the shell that launched it. So
`holt new`, `holt cd`, and `holt rm here` print a path, and a small shell
function captures that path and does the `cd` for you. Without this step those
commands still work, they just won't move you.

Add this to your `~/.zshrc` (or `~/.bashrc`), after whatever line puts Go's bin
directory on your `PATH`:

```sh
eval "$(holt shell-init zsh)"
```

Open a new terminal and you're set. The same `eval` line also wires up tab
completion for command names and worktree names.

One note for the road: that `eval` runs once per shell, at startup. If you
rebuild the binary and change what `shell-init` emits, already-open shells keep
the old function in memory. Just open a new tab and you'll have the latest.

## Quick start

```sh
cd your-project
holt init            # writes a starter holt.yml, edit it to taste
holt validate        # sanity-check the config
holt new feature-x   # create, provision, and jump into ../your-project-feature-x
# ... do your work ...
holt ls              # see all your trees and how they stack up
holt cd              # pick a tree to jump to (main is pinned at the top)
holt rm here         # tear down the current tree and head home
```

## Commands

### `holt init`

Writes a starter `holt.yml` to the repo root. It's stack-agnostic: the universal
bits are active and the framework-specific examples are commented out, ready for
you to uncomment. Use `--force` to overwrite an existing file.

### `holt validate`

Loads `holt.yml` and checks it over. Run this after editing the config to catch
problems before they bite you mid-provision.

### `holt new <branch>`

Creates a new branch and a worktree for it, then provisions the tree by running
everything in your config: copying files, linking directories, rendering env
files, allocating a port, and running your setup commands in order. When it
finishes, the shell function drops you into the new directory. Provisioning
output streams to your terminal as it goes; if anything fails, holt marks the
tree broken and leaves you where you are so you can look into it.

The branch holt forks from is recorded at creation, which is what `ls` uses to
tell you how far behind that branch you've fallen.

### `holt ls`

Lists your worktrees with name, branch, port, git status, last activity, and
state. The git column shows whether the tree is dirty and how far ahead or
behind its base branch it is, so you can spot trees that need a rebase at a
glance. Pass `--porcelain` for stable, tab-separated output meant for scripts,
or the global `--json` for machine-readable output.

### `holt cd [name]`

Prints the path to a worktree so the shell function can move you there. Give it a
name and it resolves exactly, then falls back to a fuzzy match, then a picker if
several match. Leave the name off and you get a picker over every tree, with the
main worktree pinned to the top so you can always get home. `holt cd here` is a
no-op, since you're already there.

### `holt rm [name|here]`

Removes a worktree. It runs your teardown commands, then removes the tree,
deletes its branch, prunes git's bookkeeping, and forgets it from state. Give it
a name, leave it blank to pick from a list, or use `here` to remove the tree
you're standing in. When you remove the tree you're inside, holt walks you back
to the repo root afterward so you're not stranded in a deleted directory.

Before anything destructive, the configured guards have their say (see below).

### `holt shell-init <bash|zsh|fish>`

Prints the shell function and completions described in the install section. You
normally won't call this directly; it lives in your shell config behind an
`eval`.

### `holt version`

Prints the version.

### Global flags

These work on any command:

- `--dry-run` prints what would happen and changes nothing.
- `--force` overrides guard refusals and overwrite checks.
- `--json` emits machine-readable JSON on stdout.

## Configuration

`holt init` scaffolds a documented `holt.yml`. Here's what the keys mean:

```yaml
version: 1

# Name for each worktree and its site. Default: $PROJECT-$TREE
site_name: $PROJECT-$TREE

# Where worktrees are created. Relative to the repo, or an absolute path, or ~.
# The default ".." puts them next to the repo (../<repo>-<branch>).
worktrees_dir: ..

# Branch new worktrees fork from. Leave unset to use the repo's default branch
# (origin/HEAD). Set it when your team works off something like "development".
base: development

# Gitignored files to copy from the main worktree into each new one.
copy:
  - .env

# Large, branch-invariant directories to symlink instead of copy.
link:
  - node_modules
  - vendor

# Rewrite per-worktree values into a dotenv file so trees don't collide.
env:
  - file: .env
    set:
      APP_URL: http://$SITE_NAME.test
      PORT: $PORT

# Allocate a unique port per worktree, exposed as $PORT.
port:
  range: [4000, 4999]

# Commands run in each new worktree on 'holt new', in order.
setup:
  - composer install
  - npm install
  - php artisan migrate:fresh --seed

# Commands run in a worktree on 'holt rm', before removal.
teardown:
  - herd unlink $SITE_NAME

# Warn and confirm before destructive actions.
guards:
  - uncommitted
  - unmerged
```

### Variables

You can use these in `site_name`, `env`, and your setup and teardown commands:

- `$PROJECT` is the repo name.
- `$TREE` is the branch, made filesystem-safe.
- `$SITE_NAME` is the resolved site name.
- `$PORT` is the allocated port, if you defined a `port` block.

### Guards

Guards are checks that run before `holt rm` and warn you (and ask for
confirmation) when removing a tree might lose work:

- `uncommitted` flags a dirty working tree.
- `unmerged` flags commits on the branch that aren't in the base branch.
- `unpushed` flags commits that aren't on the upstream remote.
- `stashes` flags any stash entries in the repo.

A guard's warning isn't a hard stop. You can confirm at the prompt, or pass
`--force` to skip the questions entirely.

## License

MIT
