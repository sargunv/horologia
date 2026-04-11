package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newRecurrenceCmd() *cobra.Command {
	cmd := support.GroupCommand("recurrence", "Manage task recurrence settings")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Set task recurrence"),
		support.StubCommand("clear <task>", "Clear task recurrence"),
	)
	return cmd
}
