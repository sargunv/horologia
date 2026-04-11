package usercmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var name string
	var email string
	var owner bool
	var setPassword string
	var clearPassword bool

	cmd := &cobra.Command{
		Use:   "update <user>",
		Short: "Update a user account",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.UserUpdate{}
			changed := false

			if cmd.Flags().Changed("name") {
				req.Name.SetTo(strings.TrimSpace(name))
				changed = true
			}
			if cmd.Flags().Changed("email") {
				req.Email.SetTo(strings.TrimSpace(email))
				changed = true
			}
			if cmd.Flags().Changed("owner") {
				req.IsOwner.SetTo(owner)
				changed = true
			}
			if cmd.Flags().Changed("set-password") {
				req.SetPassword.SetTo(setPassword)
				changed = true
			}
			if cmd.Flags().Changed("clear-password") {
				req.ClearPassword.SetTo(clearPassword)
				changed = true
			}

			if !changed {
				return errors.New("at least one field flag is required")
			}
			if req.SetPassword.IsSet() && req.ClearPassword.Or(false) {
				return errors.New("set-password and clear-password cannot be used together")
			}

			user, err := api.UsersUpdate(cmd.Context(), req, apigen.UsersUpdateParams{UserId: args[0]})
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

	cmd.Flags().StringVar(&name, "name", "", "Updated user name")
	cmd.Flags().StringVar(&email, "email", "", "Updated user email")
	cmd.Flags().BoolVar(&owner, "owner", false, "Updated owner status")
	cmd.Flags().StringVar(&setPassword, "set-password", "", "Set a new password")
	cmd.Flags().BoolVar(&clearPassword, "clear-password", false, "Clear the existing password")
	return cmd
}
