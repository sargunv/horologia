package spacecmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var slug string
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "update <space>",
		Short: "Update a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.SpaceUpdate{}
			changed := false

			if cmd.Flags().Changed("slug") {
				req.Slug.SetTo(strings.TrimSpace(slug))
				changed = true
			}
			if cmd.Flags().Changed("name") {
				req.Name.SetTo(strings.TrimSpace(name))
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(strings.TrimSpace(description))
				changed = true
			}

			if !changed {
				return errors.New("at least one field flag is required")
			}

			space, err := api.SpacesUpdate(cmd.Context(), req, apigen.SpacesUpdateParams{SpaceSlug: args[0]})
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

	cmd.Flags().StringVar(&slug, "slug", "", "Updated space slug")
	cmd.Flags().StringVar(&name, "name", "", "Updated space name")
	cmd.Flags().StringVar(&description, "description", "", "Updated space description")
	return cmd
}
