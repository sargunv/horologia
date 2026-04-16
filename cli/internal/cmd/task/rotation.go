package taskcmd

import (
	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newRotationCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("rotation", "Manage task rotation pools")
	cmd.AddCommand(
		newRotationSetCmd(flags),
		newRotationClearCmd(flags),
	)
	return cmd
}

func newRotationSetCmd(flags *support.RootFlags) *cobra.Command {
	var users []string

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Replace the rotation pool for a task",
		Long: `Replace the entire rotation pool for a task. Every --user flag you pass
becomes the new pool; any user not listed is removed.`,
		Example: `  # Set a two-person rotation pool
  horo task rotation set my-project SV-42 --user alice --user bob

  # Replace the pool with a single user
  horo task rotation set my-project SV-42 --user alice`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			values, err := trimRequiredStrings(users, "rotation pool member")
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{RotationPool: uniqueStrings(values)}
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

	cmd.Flags().StringArrayVar(&users, "user", nil, "User ID to include in the pool (repeatable)")
	return cmd
}

func newRotationClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task rotation pool",
		Long:  `Remove all users from the rotation pool, disabling rotation for the task.`,
		Example: `  # Remove the rotation pool
  horo task rotation clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{RotationPool: []string{}}
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
