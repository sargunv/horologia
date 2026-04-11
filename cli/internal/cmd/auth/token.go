package authcmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newTokenCmd() *cobra.Command {
	cmd := support.GroupCommand("token", "Manage personal API tokens")
	cmd.AddCommand(
		support.StubCommand("list", "List personal API tokens"),
		support.StubCommand("create", "Create a personal API token"),
		support.StubCommand("revoke <token-id>", "Revoke a personal API token"),
	)
	return cmd
}
