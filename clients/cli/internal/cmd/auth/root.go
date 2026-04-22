package authcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
)

// New builds the `horo auth` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("auth", "Manage authentication")
	cmd.GroupID = "auth"

	cmd.AddCommand(
		newLoginCmd(flags),
		newLogoutCmd(flags),
		newStatusCmd(flags),
		newTokenCmd(flags),
	)

	return cmd
}
