package cli

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "acre",
		Short:         "Spin up, manage, and tear down git worktrees",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().Bool("dry-run", false, "print the plan; perform no side effects")
	root.PersistentFlags().Bool("force", false, "override guard refusals")
	root.PersistentFlags().Bool("json", false, "emit machine-readable JSON on stdout")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newValidateCmd())

	return root
}
