# holt

A small CLI that makes git worktrees painless.

Worktrees let you check out several branches at once, each in its own directory.
The catch is that every new one needs the same setup as your main checkout (env
files, dependencies, a database, a local domain) plus a teardown when you're
done. holt handles that: describe the project once in a `holt.yml`, and `holt
new` spins up a fully provisioned worktree and drops you in, while `holt rm`
tears it down and walks you home.

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

### Sensible defaults

The scaffolded `holt.yml` is set up to follow git-worktree conventions out of
the box, so you can run `holt new` without touching the config first:

- **Sibling layout.** `worktrees_dir: ..` puts each tree next to the repo at
  `../<repo>-<branch>`, instead of nesting them inside it. This is the layout the
  [git-worktree best practices guide](https://www.gitworktree.org/guides/best-practices)
  recommends: trees stay out of the repo so your tooling, ignore rules, and file
  watchers don't trip over them, and everything for a project lives side by side.
- **Predictable names.** `site_name: $PROJECT-$TREE` names each tree after the
  repo and branch, so the directory, the site name, and the `holt ls` entry all
  match.
- **Base branch.** Unset by default, which means trees compare against the repo's
  default branch (`origin/HEAD`). Set `base` when you fork from something else,
  like `development`.
- **Safety on by default.** The `uncommitted` and `unmerged` guards are enabled,
  so `holt rm` warns before it can drop work you haven't saved or merged.

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

# Rewrite per-worktree values into a dotenv file so trees don't collide.
env:
  - file: .env
    set:
      PORT: $PORT
      BASE_URL: http://localhost:$PORT

# Allocate a unique port per worktree, exposed as $PORT.
port:
  range: [3000, 3999]

# Commands run in each new worktree on 'holt new', in order.
setup:
  - go mod download
  - npm install
  - docker compose -p $SITE_NAME up -d

# Commands run in a worktree on 'holt rm', before removal.
teardown:
  - docker compose -p $SITE_NAME down

# Warn and confirm before destructive actions.
guards:
  - uncommitted
  - unmerged
```

### Variables

You can use these in `site_name`, `env`, and your setup and teardown commands.
Say your repo lives at `/Users/you/dev/acme`, you run `holt new feature/login`,
and the `port` block hands out `3000`. The variables then resolve to:

| Variable      | Example value                          | What it is                                          |
| ------------- | -------------------------------------- | --------------------------------------------------- |
| `$REPO_ROOT`  | `/Users/you/dev/acme`                  | Absolute path to the main repo.                     |
| `$PROJECT`    | `acme`                                 | The repo's directory name.                          |
| `$BRANCH`     | `feature/login`                        | The branch, exactly as you typed it.                |
| `$TREE`       | `feature-login`                        | The branch made filesystem-safe (`/` becomes `-`).  |
| `$SITE_NAME`  | `acme-feature-login`                   | The resolved site name (`$PROJECT-$TREE` default).  |
| `$WORKTREE`   | `/Users/you/dev/acme-feature-login`    | Absolute path to the new worktree.                  |
| `$PORT`       | `3000`                                 | The allocated port (only if a `port` block is set). |

So with the `env` block shown above, the worktree's `.env` would be written
with the values filled in:

```sh
PORT=3000
BASE_URL=http://localhost:3000
```

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
