package spacecmd

import (
	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
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
		Long:  `List all task priority levels configured for a space. Results appear in the configured display order.`,
		Example: `  # List priority levels
  horo space priority list my-project`,
		Args: cobra.ExactArgs(1),
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
		Long: `Replace the full ordered list of task priority levels for a space. This
removes every existing level and writes the provided set. Tasks that
reference a removed level will lose that value. Pass each level with
a separate --name flag; flag order sets display order.`,
		Example: `  # Set four priority levels
  horo space priority replace my-project \
    --name Low --name Medium --name High --name Critical

  # Clear all priority levels
  horo space priority replace my-project`,
		Args: cobra.ExactArgs(1),
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

	cmd.Flags().StringArrayVar(&names, "name", nil, "Priority level name (repeatable)")
	return cmd
}
