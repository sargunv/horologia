package recipecmd

import (
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newSearchCmd(flags *support.RootFlags) *cobra.Command {
	var spaceSlug string
	var limit int32
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search visible recipes across spaces",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			params := apigen.RecipesSearchParams{Q: strings.TrimSpace(args[0])}
			if cmd.Flags().Changed("space") {
				params.SpaceSlug.SetTo(strings.TrimSpace(spaceSlug))
			}
			if limit > 0 {
				params.Limit.SetTo(limit)
			}
			resp, err := api.RecipesSearch(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(resp)
			}
			return printRecipeList(app, resp.Items)
		}),
	}
	cmd.Flags().StringVar(&spaceSlug, "space", "", "Restrict results to a single space slug")
	cmd.Flags().Int32Var(&limit, "limit", 10, "Maximum number of items to return")
	return cmd
}
