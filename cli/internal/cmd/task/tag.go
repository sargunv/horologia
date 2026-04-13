package taskcmd

import (
	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
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
		Long: `Replace the full tag list on a task. Every --tag flag you pass becomes
the new list; any tag not included is removed. To add or remove a single
tag without affecting others, use add or remove.`,
		Example: `  # Set a single tag
  horo task tag set my-project SV-42 --tag urgent

  # Set multiple tags
  horo task tag set my-project SV-42 --tag urgent --tag backend`,
		Args: cobra.ExactArgs(2),
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

	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag name (repeatable)")
	return cmd
}

func newTagAddCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <space> <task> <tag>",
		Short: "Add a tag to a task",
		Long: `Add a tag to the task. If the tag is already present, the command
succeeds with no change.`,
		Example: `  # Add the "urgent" tag
  horo task tag add my-project SV-42 urgent`,
		Args: cobra.ExactArgs(3),
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
		Long: `Remove a tag from the task. If the tag is not present, the command
succeeds with no change.`,
		Example: `  # Remove the "urgent" tag
  horo task tag remove my-project SV-42 urgent`,
		Args: cobra.ExactArgs(3),
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
		Long:  `Remove all tags from a task, leaving the tag list empty.`,
		Example: `  # Remove all tags
  horo task tag clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
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
