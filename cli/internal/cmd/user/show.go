package usercmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newShowCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <user>",
		Short: "Show a user",
		Long:  `Display a user's name, email, owner status, and password configuration.`,
		Example: `  # Inspect a user's profile
  horo user show alice`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			user, err := api.UsersGet(cmd.Context(), apigen.UsersGetParams{UserId: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(user)
			}

			printUser(app, user)
			return nil
		}),
	}
}
