package spacecmd

import (
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var slug string
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a space",
		Long: `Create a new space with the given slug and name. The slug must be
unique across all spaces and is used to reference the space in other commands.`,
		Example: `  # Create a space
  horo space create --slug my-project --name "My Project"

  # Create a space with a description
  horo space create --slug my-project --name "My Project" \
    --description "Tracks all project tasks"`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.SpaceCreate{
				Slug: strings.TrimSpace(slug),
				Name: strings.TrimSpace(name),
			}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(strings.TrimSpace(description))
			}

			space, err := api.SpacesCreate(cmd.Context(), req)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(space)
			}

			printSpace(app, space)
			return nil
		}),
	}

	cmd.Flags().StringVar(&slug, "slug", "", "Space slug")
	cmd.Flags().StringVar(&name, "name", "", "Space name")
	cmd.Flags().StringVar(&description, "description", "", "Space description")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
