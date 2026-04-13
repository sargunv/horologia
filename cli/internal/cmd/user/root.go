package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
)

// New builds the `horo user` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("user", "Inspect and manage user accounts")
	cmd.GroupID = "account"

	cmd.AddCommand(
		newMeCmd(flags),
		newShowCmd(flags),
		newUpdateCmd(flags),
		newTasksCmd(flags),
		newActivityCmd(flags),
		newListCmd(flags),
		newCreateCmd(flags),
		newDeleteCmd(flags),
	)

	return cmd
}
