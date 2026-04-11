package usercmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newTasksCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "tasks <user>",
		Short: "List tasks assigned to a user",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			params := apigen.UserTasksListParams{UserId: args[0]}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)

			resp, err := api.UserTasksList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			printTaskPage(app, resp)
			return nil
		}),
	}

	addPageFlags(cmd, &page)
	return cmd
}
