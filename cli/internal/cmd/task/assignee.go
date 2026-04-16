package taskcmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newAssigneeCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("assignee", "Manage task assignees")
	cmd.AddCommand(
		newAssigneeSetCmd(flags),
		newAssigneeAddCmd(flags),
		newAssigneeRemoveCmd(flags),
		newAssigneeClearCmd(flags),
	)
	return cmd
}

func newAssigneeSetCmd(flags *support.RootFlags) *cobra.Command {
	var users []string

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Replace task assignees",
		Long: `Replace the full assignee list on a task. Every --user flag you pass
becomes the new list; any user not included is removed. To add or
remove a single assignee without affecting others, use add or remove.`,
		Example: `  # Assign a single user
  horo task assignee set my-project SV-42 --user alice

  # Assign multiple users
  horo task assignee set my-project SV-42 --user alice --user bob`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			values, err := trimRequiredStrings(users, "assignee")
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{AssigneeIds: uniqueStrings(values)}
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

	cmd.Flags().StringArrayVar(&users, "user", nil, "Assignee user ID (repeatable)")
	return cmd
}

func newAssigneeAddCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <space> <task> <user>",
		Short: "Add a task assignee",
		Long: `Add a user to the task's assignee list. If the user is already
assigned, the command succeeds with no change.`,
		Example: `  # Add alice as an assignee
  horo task assignee add my-project SV-42 alice`,
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

			req := &apigen.TaskUpdate{AssigneeIds: uniqueStrings(append(append([]string{}, task.AssigneeIds...), args[2]))}
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

func newAssigneeRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <task> <user>",
		Short: "Remove a task assignee",
		Long: `Remove a user from the task's assignee list. If the user is not
currently assigned, the command succeeds with no change.`,
		Example: `  # Remove alice from the assignees
  horo task assignee remove my-project SV-42 alice`,
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

			req := &apigen.TaskUpdate{AssigneeIds: withoutString(task.AssigneeIds, args[2])}
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

func newAssigneeClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear task assignees",
		Long:  `Remove all assignees from a task, leaving the assignee list empty.`,
		Example: `  # Remove all assignees
  horo task assignee clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{AssigneeIds: []string{}}
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
