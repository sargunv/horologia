package usercmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newDeleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <user>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			if err := api.UsersDelete(cmd.Context(), apigen.UsersDeleteParams{UserId: args[0]}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"userId":  args[0],
					"deleted": true,
				})
			}

			app.Printf("Deleted user %s\n", args[0])
			return nil
		}),
	}
}
