package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newRelationCmd() *cobra.Command {
	cmd := support.GroupCommand("relation", "Manage task relations")
	cmd.AddCommand(
		support.StubCommand("add <task> <kind> <related-task>", "Add a relation to a task"),
		support.StubCommand("remove <task> <kind> <related-task>", "Remove a relation from a task"),
	)
	return cmd
}
