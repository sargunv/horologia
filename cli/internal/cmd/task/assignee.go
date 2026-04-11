package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newAssigneeCmd() *cobra.Command {
	cmd := support.GroupCommand("assignee", "Manage task assignees")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Replace task assignees"),
		support.StubCommand("add <task> <user>", "Add a task assignee"),
		support.StubCommand("remove <task> <user>", "Remove a task assignee"),
		support.StubCommand("clear <task>", "Clear task assignees"),
	)
	return cmd
}
