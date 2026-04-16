package usercmd

import (
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var name string
	var email string
	var owner bool
	var password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		Long: `Create a new user. The --name and --email flags are required.
Pass --owner to grant the new user owner privileges. Pass --password
to set an initial password; if omitted, the user has no password.`,
		Example: `  # Create a basic user
  horo user create --name "Alice Ng" --email alice@example.com

  # Create an owner with an initial password
  horo user create --name "Bob Li" --email bob@example.com \
    --owner --password "$INITIAL_PASSWORD"`,
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
