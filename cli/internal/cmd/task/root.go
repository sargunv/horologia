package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
)

// New builds the `horo task` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("task", "Inspect and manage tasks")
	cmd.GroupID = "workspace"

	cmd.AddCommand(
		newReadWriteCmds(flags)...,
	)
	cmd.AddCommand(
		newSearchCmd(flags),
		newDueCmd(flags),
		newAssigneeCmd(flags),
		newTagCmd(flags),
		newRecurrenceCmd(flags),
		newOverdueActionCmd(flags),
		newRotationCmd(flags),
		newRelationCmd(flags),
	)

	return cmd
}
