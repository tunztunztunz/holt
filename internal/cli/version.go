package cli

import "github.com/spf13/cobra"

const Version = "0.1.0-dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the acre version",
		RunE: func(cmd *cobra.Command, args []string) error {
			resultf("%s\n", Version)
			return nil
		},
	}
}
