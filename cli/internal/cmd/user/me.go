package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newMeCmd(flags *support.RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated Tend user",
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			if app.API == nil {
				return runtime.MissingServerError()
			}

			user, err := app.API.UsersMe(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(user)
			}

			app.Printf("Name:        %s\n", user.Name)
			app.Printf("Email:       %s\n", user.Email)
			app.Printf("ID:          %s\n", user.ID)
			app.Printf("Owner:       %t\n", user.IsOwner)
			app.Printf("HasPassword: %t\n", user.HasPassword)
			return nil
		}),
	}

	return cmd
}
