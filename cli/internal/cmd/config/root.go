package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend config` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("config", "Manage CLI configuration")
	cmd.GroupID = "foundation"
	cmd.RunE = support.DefaultSubcommand("show")

	cmd.AddCommand(
		newShowCmd(flags),
		newSetCmd(flags),
		newUnsetCmd(flags),
		newPathCmd(flags),
	)

	return cmd
}
