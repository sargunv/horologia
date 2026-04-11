package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Example: `  # See all users on the server
  tend user list

  # Get machine-readable output
  tend user list --json`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			users, err := api.UsersList(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(users)
			}

			return printUserList(app, users.Items)
		}),
	}
}
