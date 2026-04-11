package spacecmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newDeleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <space>",
		Short: "Delete a space",
		Long: `Permanently delete a space and all its tasks, tags, and configuration.
This cannot be undone.`,
		Example: `  # Permanently remove a space and all its data
  tend space delete my-project`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			if err := api.SpacesDelete(cmd.Context(), apigen.SpacesDeleteParams{SpaceSlug: args[0]}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"spaceSlug": args[0],
					"deleted":   true,
				})
			}

			app.Printf("Deleted space %s\n", args[0])
			return nil
		}),
	}
}
