package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accessible spaces",
		Long:  `List all spaces where you are a member, including each space's slug, name, and description.`,
		Example: `  # See which spaces you belong to
  horo space list

  # Get machine-readable output
  horo space list --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			spaces, err := api.SpacesList(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(spaces)
			}

			return printSpaceList(app, spaces.Items)
		}),
	}
}
