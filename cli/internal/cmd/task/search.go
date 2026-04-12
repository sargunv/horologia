package taskcmd

import (
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newSearchCmd(flags *support.RootFlags) *cobra.Command {
	var spaceSlug string
	var excludeTaskID string
	var limit int32

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search visible tasks across spaces",
		Long: `Search tasks visible to the authenticated user across all spaces.

Use --space to restrict results to a single space, and --exclude-task to omit
one task from the results when using the command as a picker helper.`,
		Example: `  # Search all visible tasks
  tend task search "login"

  # Search within a single space
  tend task search "login" --space app

  # Exclude the current task from results
  tend task search "plan" --exclude-task T42`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			params := apigen.TasksSearchParams{
				Q: strings.TrimSpace(args[0]),
			}
			if cmd.Flags().Changed("space") {
				params.SpaceSlug.SetTo(strings.TrimSpace(spaceSlug))
			}
			if cmd.Flags().Changed("exclude-task") {
				params.ExcludeTaskId.SetTo(strings.TrimSpace(excludeTaskID))
			}
			if limit > 0 {
				params.Limit.SetTo(limit)
			}

			resp, err := api.TasksSearch(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskSearchResultTable(app, resp.Items)
		}),
	}

	cmd.Flags().StringVar(&spaceSlug, "space", "", "Restrict results to a single space slug")
	cmd.Flags().StringVar(&excludeTaskID, "exclude-task", "", "Exclude a task ID from the results")
	cmd.Flags().Int32Var(&limit, "limit", 10, "Maximum number of items to return")
	return cmd
}
