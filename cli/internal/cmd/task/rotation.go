package taskcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newRotationCmd() *cobra.Command {
	cmd := support.GroupCommand("rotation", "Manage task rotation pools")
	cmd.AddCommand(
		support.StubCommand("set <task>", "Set a task rotation pool"),
		support.StubCommand("clear <task>", "Clear a task rotation pool"),
	)
	return cmd
}
