package taskcmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newTagCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("tag", "Manage task tags")
	cmd.AddCommand(
		newTagSetCmd(flags),
		newTagAddCmd(flags),
		newTagRemoveCmd(flags),
		newTagClearCmd(flags),
	)
	return cmd
}

func newTagSetCmd(flags *support.RootFlags) *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Replace task tags",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			values, err := trimRequiredStrings(tags, "tag")
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{Tags: uniqueStrings(values)}
			task, err := api.SpaceTasksUpdate(cmd.Context(), req, apigen.SpaceTasksUpdateParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(task)
			}
			printTask(app, task)
			return nil
		}),
	}

	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag name; repeat to set the full tag list")
	return cmd
}

func newTagAddCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <space> <task> <tag>",
		Short: "Add a tag to a task",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			task, err := readTask(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}

			req := &apigen.TaskUpdate{Tags: uniqueStrings(append(append([]string{}, task.Tags...), args[2]))}
			updated, err := api.SpaceTasksUpdate(cmd.Context(), req, apigen.SpaceTasksUpdateParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(updated)
			}
			printTask(app, updated)
			return nil
		}),
	}
}

func newTagRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <task> <tag>",
		Short: "Remove a tag from a task",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			task, err := readTask(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}

			req := &apigen.TaskUpdate{Tags: withoutString(task.Tags, args[2])}
			updated, err := api.SpaceTasksUpdate(cmd.Context(), req, apigen.SpaceTasksUpdateParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(updated)
			}
			printTask(app, updated)
			return nil
		}),
	}
}

func newTagClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear task tags",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{Tags: []string{}}
			task, err := api.SpaceTasksUpdate(cmd.Context(), req, apigen.SpaceTasksUpdateParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(task)
			}
			printTask(app, task)
			return nil
		}),
	}
}
