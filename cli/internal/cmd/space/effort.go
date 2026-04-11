package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newEffortCmd() *cobra.Command {
	cmd := support.GroupCommand("effort", "Manage task effort levels for a space")
	cmd.AddCommand(
		support.StubCommand("list <space>", "List task effort levels in a space"),
		support.StubCommand("replace <space>", "Replace task effort levels in a space"),
	)
	return cmd
}
