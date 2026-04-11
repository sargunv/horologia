package authcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend auth` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("auth", "Authenticate and manage local credentials")
	cmd.GroupID = "auth"

	cmd.AddCommand(
		support.StubCommand("login", "Authenticate with a Tend server"),
		support.StubCommand("logout", "Clear local authentication state"),
		support.StubCommand("status", "Show the current authentication state"),
		newTokenCmd(),
	)

	return cmd
}
