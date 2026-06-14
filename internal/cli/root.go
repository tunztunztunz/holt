package cli

type Globals struct {
	DryRun bool `help:"Print the plan; perform no side effects."`
	Force  bool `help:"Override guard refusals."`
	JSON   bool `help:"Emit machine-readable JSON on stdout."`
}

type CLI struct {
	Globals

	ShellInit ShellInitCmd `cmd:"" help:"Print shell integration (eval in your rc file)."`
	Version   versionCmd   `cmd:"" help:"Print the holt version."`
	Init      initCmd      `cmd:"" help:"Scaffold holt.yml in the repo root."`
	Validate  validateCmd  `cmd:"" help:"Load and validate holt.yml."`
	New       newCmd       `cmd:"" help:"Create and provision a worktree."`
	Ls        LsCmd        `cmd:"" help:"List worktrees with status."`
	Cd        CdCmd        `cmd:"" help:"Print a worktree's path (for shell cd)."`
	Rm        RmCmd        `cmd:"" help:"Remove a worktree."`
}
