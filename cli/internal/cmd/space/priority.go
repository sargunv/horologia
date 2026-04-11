package spacecmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newPriorityCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("priority", "Manage task priority levels for a space")
	cmd.AddCommand(
		newPriorityListCmd(flags),
		newPriorityReplaceCmd(flags),
	)
	return cmd
}

func newPriorityListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space>",
		Short: "List task priority levels in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.SpaceTaskPriorityLevelsList(cmd.Context(), apigen.SpaceTaskPriorityLevelsListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskPriorityLevelList(app, resp.Items)
		}),
	}
}

func newPriorityReplaceCmd(flags *support.RootFlags) *cobra.Command {
	var names []string

	cmd := &cobra.Command{
		Use:   "replace <space>",
		Short: "Replace task priority levels in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			values, err := trimOptionalValues(names, "priority level")
			if err != nil {
				return err
			}

			items := make([]apigen.TaskPriorityLevelInput, 0, len(values))
			for _, name := range values {
				items = append(items, apigen.TaskPriorityLevelInput{Name: name})
			}

			resp, err := api.SpaceTaskPriorityLevelsReplace(cmd.Context(), &apigen.TaskPriorityLevelReplace{
				Items: items,
			}, apigen.SpaceTaskPriorityLevelsReplaceParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskPriorityLevelList(app, resp.Items)
		}),
	}

	cmd.Flags().StringArrayVar(&names, "name", nil, "Priority level name; repeat to set the full ordered list")
	return cmd
}
