package cli

const Version = "0.1.0-dev"

type versionCmd struct{}

func (c *versionCmd) Run() error {
	resultf("%s\n", Version)
	return nil
}
