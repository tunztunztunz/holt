package cli

import (
	"errors"
	"os"

	"github.com/alecthomas/kong"
)

func Execute() {
	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("acre"),
		kong.Description("Spin up, manage, and tear down git worktrees."),
	)

	err := kctx.Run(&cli.Globals)
	if err == nil {
		os.Exit(ExitOK)
	}

	if ee, ok := errors.AsType[*ExitError](err); ok {
		warnf("%s", ee.Error())
		os.Exit(ee.Code)
	}
	warnf("%s", err.Error())
	os.Exit(ExitRuntime)
}
