package cli

// zshBashInit shadows the `holt` binary with a function. `new`, `cd`, `rm`, and
// `harvest` can emit a worktree path on stdout for the shell to cd into; the
// function captures stdout and, only if it's a real directory, cds there.
// Anything else on stdout (e.g. `--help`) is printed through rather than treated
// as a path, and a non-zero exit passes the captured output and status straight
// back. Human output — prompts, plans, provisioning — goes to stderr, so it
// streams live regardless. Works in both bash and zsh.
const zshBashInit = `holt() {
	case "$1" in
	new|cd|rm|harvest)
		local out code
		out="$(command holt "$@")"; code=$?
		if [ "$code" -ne 0 ]; then
			[ -n "$out" ] && printf '%s\n' "$out"
			return "$code"
		fi
		if [ -d "$out" ]; then
			builtin cd "$out"
		elif [ -n "$out" ]; then
			printf '%s\n' "$out"
		fi ;;
	*)
		command holt "$@" ;;
	esac
}`

// Completions are hand-emitted. The subcommand list is static; the worktree names for
// `cd`/`rm` are dynamic, pulled from `holt ls --porcelain` (first column) so
// they always match the live, reconciled set. Keep the static list in sync with
// the commands on root.go's CLI struct.

const bashCompletion = `_holt() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W "version init validate new ls cd rm harvest" -- "$cur"))
  elif [ "${COMP_WORDS[1]}" = "cd" ] || [ "${COMP_WORDS[1]}" = "rm" ] || [ "${COMP_WORDS[1]}" = "harvest" ]; then
    COMPREPLY=($(compgen -W "$(command holt ls --porcelain | cut -f1)" -- "$cur"))
  fi
}
complete -F _holt holt`

const zshCompletion = `_holt() {
  if (( CURRENT == 2 )); then
    compadd version init validate new ls cd rm harvest
  elif [[ "$words[2]" == (cd|rm|harvest) ]]; then
    compadd -- ${(f)"$(command holt ls --porcelain | cut -f1)"}
  fi
}
compdef _holt holt`

// ShellInitCmd is `holt shell-init <bash|zsh|fish>`. The `enum` tag makes Kong
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
