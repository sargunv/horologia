package spacecmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newShowCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <space>",
		Short: "Show a space",
		Long:  `Display a space's slug, name, description, and membership summary.`,
		Example: `  # Inspect a space by slug
  horo space show my-project`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			space, err := api.SpacesRead(cmd.Context(), apigen.SpacesReadParams{SpaceSlug: args[0]})
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
}
