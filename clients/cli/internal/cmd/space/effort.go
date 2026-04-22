package spacecmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newEffortCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("effort", "Manage task effort levels for a space")
	cmd.AddCommand(
		newEffortListCmd(flags),
		newEffortReplaceCmd(flags),
	)
	return cmd
}

func newEffortListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space>",
		Short: "List task effort levels in a space",
		Long:  `List all task effort levels configured for a space. Results appear in the configured display order.`,
		Example: `  # List effort levels
  horo space effort list my-project`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.SpaceTaskEffortLevelsList(cmd.Context(), apigen.SpaceTaskEffortLevelsListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskEffortLevelList(app, resp.Items)
		}),
	}
}

func newEffortReplaceCmd(flags *support.RootFlags) *cobra.Command {
	var names []string

	cmd := &cobra.Command{
		Use:   "replace <space>",
		Short: "Replace task effort levels in a space",
		Long: `Replace the full ordered list of task effort levels for a space. This
removes every existing level and writes the provided set. Tasks that
reference a removed level will lose that value. Pass each level with
a separate --name flag; flag order sets display order.`,
		Example: `  # Set three effort levels
  horo space effort replace my-project \
    --name Small --name Medium --name Large

  # Clear all effort levels
  horo space effort replace my-project`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			values, err := trimOptionalValues(names, "effort level")
			if err != nil {
				return err
			}

			items := make([]apigen.TaskEffortLevelInput, 0, len(values))
			for _, name := range values {
				items = append(items, apigen.TaskEffortLevelInput{Name: name})
			}

			resp, err := api.SpaceTaskEffortLevelsReplace(cmd.Context(), &apigen.TaskEffortLevelReplace{
				Items: items,
			}, apigen.SpaceTaskEffortLevelsReplaceParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskEffortLevelList(app, resp.Items)
		}),
	}

	cmd.Flags().StringArrayVar(&names, "name", nil, "Effort level name (repeatable)")
	return cmd
}
