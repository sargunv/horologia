package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
)

// New builds the `horo space` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("space", "Inspect and manage spaces")
	cmd.GroupID = "workspace"

	cmd.AddCommand(
		newListCmd(flags),
		newShowCmd(flags),
		newCreateCmd(flags),
		newUpdateCmd(flags),
		newDeleteCmd(flags),
		newActivityCmd(flags),
	)
	cmd.AddCommand(
		newMemberCmd(flags),
		newTagCmd(flags),
		newStatusCmd(flags),
		newEffortCmd(flags),
		newPriorityCmd(flags),
	)

	return cmd
}
