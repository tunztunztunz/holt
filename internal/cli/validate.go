package cli

import (
	"github.com/tunztunztunz/acre/internal/config"
	"github.com/tunztunztunz/acre/internal/gitx"
)

type validateCmd struct{}

func (c *validateCmd) Run() error {
	root, err := gitx.RepoRoot()
	if err != nil {
		return Exitf(ExitUsage, "%v", err)
	}
	p, err := config.Load(root) // unknown key -> error here
	if err != nil {
		return Exitf(ExitUsage, "%v", err) // ExitUsage == 2
	}
	if err := p.Validate(); err != nil { // bad values -> error here
		return Exitf(ExitUsage, "%v", err)
	}
	infof("acre.yml is valid")
	return nil
}
