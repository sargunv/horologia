package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
)

func newMemberCmd() *cobra.Command {
	cmd := support.GroupCommand("member", "Manage space membership and roles")
	cmd.AddCommand(
		support.StubCommand("list <space>", "List members of a space"),
		support.StubCommand("add <space> <user>", "Add a user to a space"),
		support.StubCommand("set-role <space> <user> <role>", "Change a member role"),
		support.StubCommand("remove <space> <user>", "Remove a member from a space"),
	)
	return cmd
}
