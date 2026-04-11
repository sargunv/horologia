package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

// New builds the `tend user` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("user", "Inspect and manage user accounts")
	cmd.GroupID = "account"

	cmd.AddCommand(
		newMeCmd(flags),
		support.StubCommand("show <user>", "Show a user account"),
		support.StubCommand("update <user>", "Update a user account"),
		support.StubCommand("tasks <user>", "List tasks assigned to a user"),
		support.StubCommand("activity <user>", "Show activity for a user"),
		support.StubCommand("list", "List users"),
		support.StubCommand("create", "Create a user"),
		support.StubCommand("delete <user>", "Delete a user"),
	)

	return cmd
}
