package spacecmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newStatusCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("status", "Manage task statuses for a space")
	cmd.AddCommand(
		newStatusListCmd(flags),
		newStatusReplaceCmd(flags),
	)
	return cmd
}

func newStatusListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space>",
		Short: "List task statuses in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.SpaceTaskStatusesList(cmd.Context(), apigen.SpaceTaskStatusesListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskStatusList(app, resp.Items)
		}),
	}
}

func newStatusReplaceCmd(flags *support.RootFlags) *cobra.Command {
	var initial string
	var intermediate []string
	var completion []string

	cmd := &cobra.Command{
		Use:   "replace <space>",
		Short: "Replace task statuses in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			initialValues, err := trimRequiredValues([]string{initial}, "initial status")
			if err != nil {
				return err
			}
			completionValues, err := trimRequiredValues(completion, "completion status")
			if err != nil {
				return err
			}
			intermediateValues, err := trimOptionalValues(intermediate, "intermediate status")
			if err != nil {
				return err
			}

			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			items := make([]apigen.TaskStatusInput, 0, len(initialValues)+len(intermediateValues)+len(completionValues))
			for _, name := range initialValues {
				items = append(items, apigen.TaskStatusInput{Name: name, Category: apigen.TaskStatusCategoryInitial})
			}
			for _, name := range intermediateValues {
				items = append(items, apigen.TaskStatusInput{Name: name, Category: apigen.TaskStatusCategoryIntermediate})
			}
			for _, name := range completionValues {
				items = append(items, apigen.TaskStatusInput{Name: name, Category: apigen.TaskStatusCategoryCompletion})
			}

			resp, err := api.SpaceTaskStatusesReplace(cmd.Context(), &apigen.TaskStatusReplace{
				Items: items,
			}, apigen.SpaceTaskStatusesReplaceParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskStatusList(app, resp.Items)
		}),
	}

	cmd.Flags().StringVar(&initial, "initial", "", "Initial status name")
	cmd.Flags().StringArrayVar(&intermediate, "intermediate", nil, "Intermediate status name; repeat to add more than one")
	cmd.Flags().StringArrayVar(&completion, "completion", nil, "Completion status name; repeat to add more than one")
	_ = cmd.MarkFlagRequired("initial")
	return cmd
}
