package spacecmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accessible spaces",
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
