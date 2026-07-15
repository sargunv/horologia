package recipecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
)

// New builds the `horo recipe` command tree.
func New(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("recipe", "Inspect and manage recipes")
	cmd.GroupID = "workspace"

	cmd.AddCommand(
		newListCmd(flags),
		newSearchCmd(flags),
		newShowCmd(flags),
		newCreateCmd(flags),
		newUpdateCmd(flags),
		newDeleteCmd(flags),
		newActivityCmd(flags),
		newTagCmd(flags),
		newIngredientCmd(flags),
		newInstructionCmd(flags),
	)

	return cmd
}
