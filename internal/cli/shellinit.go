package cli

// zshBashInit shadows the `acre` binary with a function: `acre cd X` captures
// the path the real binary prints and `cd`s to it; anything else passes through
// to `command acre`. Works in both bash and zsh.
const zshBashInit = `acre() {
	if [ "$1" = "cd" ]; then
		local dir
		dir="$(command acre cd "${@:2}")" || return $?
		builtin cd "$dir"
	else
		command acre "$@"
	fi
}`

// Completions are hand-emitted. The subcommand list is static; the worktree names for
// `cd`/`rm` are dynamic, pulled from `acre ls --porcelain` (first column) so
// they always match the live, reconciled set. Keep the static list in sync with
// the commands on root.go's CLI struct.

const bashCompletion = `_acre() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W "version init validate new ls cd rm" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "cd" ] || [ "${COMP_WORDS[1]}" = "rm" ]; then
    COMPREPLY=($(compgen -W "$(command acre ls --porcelain | cut -f1)" -- "$cur"))
  fi
}
complete -F _acre acre`

const zshCompletion = `_acre() {
  if (( CURRENT == 2 )); then
    compadd version init validate new ls cd rm
  elif [[ "$words[2]" == (cd|rm) ]]; then
    compadd -- ${(f)"$(command acre ls --porcelain | cut -f1)"}
  fi
}
compdef _acre acre`

// ShellInitCmd is `acre shell-init <bash|zsh|fish>`. The `enum` tag makes Kong
// reject an unknown shell at parse time with a usage error — no manual default
// case needed.
type ShellInitCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell to emit integration for."`
}

func (c *ShellInitCmd) Run() error {
	switch c.Shell {
	case "bash":
		resultf("%s\n\n%s\n", zshBashInit, bashCompletion)
	case "zsh":
		resultf("%s\n\n%s\n", zshBashInit, zshCompletion)
	case "fish":
		return Exitf(ExitUsage, "fish: not in prototype yet")
	}
	return nil
}
