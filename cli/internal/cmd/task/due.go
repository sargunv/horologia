package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newDueCmd() *cobra.Command {
	cmd := support.GroupCommand("due", "Manage task due dates")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Set a task due date"),
		support.StubCommand("clear <task>", "Clear a task due date"),
	)
	return cmd
}
