package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend task` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("task", "Inspect and manage tasks")
	cmd.GroupID = "workspace"

	cmd.AddCommand(
		newReadWriteCmds()...,
	)
	cmd.AddCommand(
		newDueCmd(),
		newAssigneeCmd(),
		newTagCmd(),
		newRecurrenceCmd(),
		newOverdueActionCmd(),
		newRotationCmd(),
		newRelationCmd(),
	)

	return cmd
}
