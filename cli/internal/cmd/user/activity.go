package usercmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newActivityCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "activity <user>",
		Short: "Show activity for a user",
		Args:  cobra.ExactArgs(1),
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
