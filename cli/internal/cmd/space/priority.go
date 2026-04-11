package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newPriorityCmd() *cobra.Command {
	cmd := support.GroupCommand("priority", "Manage task priority levels for a space")
	cmd.AddCommand(
		support.StubCommand("list <space>", "List task priority levels in a space"),
		support.StubCommand("replace <space>", "Replace task priority levels in a space"),
	)
	return cmd
}
