package cli

import "github.com/tunztunztunz/holt/internal/config"

type validateCmd struct{}

func (c *validateCmd) Run(p *config.Profile) error {
	if err := p.Validate(); err != nil {
		return Exitf(ExitUsage, "%v", err)
	}
	infof("holt.yml is valid")
	return nil
}
