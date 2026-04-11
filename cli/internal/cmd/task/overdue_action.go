package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newOverdueActionCmd() *cobra.Command {
	cmd := support.GroupCommand("overdue-action", "Manage task overdue actions")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Set a task overdue action"),
		support.StubCommand("clear <task>", "Clear a task overdue action"),
	)
	return cmd
}
