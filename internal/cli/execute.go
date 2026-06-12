package cli

import (
	"errors"
	"os"
)

func Execute() {
	err := newRootCmd().Execute()
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
