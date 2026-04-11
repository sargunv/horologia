package spacecmd

import (
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var slug string
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a space",
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
