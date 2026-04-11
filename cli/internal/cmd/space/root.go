package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend space` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("space", "Inspect and manage spaces")
	cmd.GroupID = "workspace"

	cmd.AddCommand(
		newCRUDCmds()...,
	)
	cmd.AddCommand(
		newMemberCmd(),
		newTagCmd(),
		newStatusCmd(),
		newEffortCmd(),
		newPriorityCmd(),
	)

	return cmd
}
