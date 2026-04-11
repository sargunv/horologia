package authcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend auth` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("auth", "Manage authentication")
	cmd.GroupID = "auth"

	cmd.AddCommand(
		support.StubCommand("login", "Log in to a Tend server (not yet implemented)"),
		support.StubCommand("logout", "Log out and clear credentials (not yet implemented)"),
		newStatusCmd(flags),
		newTokenCmd(flags),
	)

	return cmd
}
