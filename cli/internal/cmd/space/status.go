package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newStatusCmd() *cobra.Command {
	cmd := support.GroupCommand("status", "Manage task statuses for a space")
	cmd.AddCommand(
		support.StubCommand("list <space>", "List task statuses in a space"),
		support.StubCommand("replace <space>", "Replace task statuses in a space"),
	)
	return cmd
}
