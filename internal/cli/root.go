package cli

type Globals struct {
	DryRun bool `help:"Print the plan; perform no side effects."`
	Force  bool `help:"Override guard refusals."`
	JSON   bool `help:"Emit machine-readable JSON on stdout."`
}

type CLI struct {
	Globals

	Version  versionCmd  `cmd:"" help:"Print the acre version."`
	Init     initCmd     `cmd:"" help:"Scaffold acre.yml in the repo root."`
	Validate validateCmd `cmd:"" help:"Load and validate acre.yml."`
	New      newCmd      `cmd:"" help:"Create and provision a worktree."`
}
