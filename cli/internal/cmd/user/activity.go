package usercmd

import (
	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newActivityCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "activity <user>",
		Short: "Show activity for a user",
		Long: `Show the activity log for a user. Results are paginated; use --cursor
with the value from a previous response to fetch the next page.`,
		Example: `  # Show activity for a user
  horo user activity alice

  # Limit to 5 entries
  horo user activity alice --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			params := apigen.UserActivityListParams{UserId: args[0]}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)

			resp, err := api.UserActivityList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			printActivityPage(app, resp)
			return nil
		}),
	}

	addPageFlags(cmd, &page)
	return cmd
}
