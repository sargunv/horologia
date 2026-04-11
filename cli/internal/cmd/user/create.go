package usercmd

import (
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var name string
	var email string
	var owner bool
	var password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.UserCreate{
				Name:  strings.TrimSpace(name),
				Email: strings.TrimSpace(email),
			}
			if cmd.Flags().Changed("owner") {
				req.IsOwner.SetTo(owner)
			}
			if strings.TrimSpace(password) != "" {
				req.Password.SetTo(password)
			}

			user, err := api.UsersCreate(cmd.Context(), req)
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

	cmd.Flags().StringVar(&name, "name", "", "User name")
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().BoolVar(&owner, "owner", false, "Create the user as an owner")
	cmd.Flags().StringVar(&password, "password", "", "Initial password")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}
