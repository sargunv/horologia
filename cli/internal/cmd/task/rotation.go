package taskcmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
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
		Short: "Set a task rotation pool",
		Args:  cobra.ExactArgs(2),
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

	cmd.Flags().StringArrayVar(&users, "user", nil, "Rotation pool user ID; repeat to set the full pool")
	return cmd
}

func newRotationClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task rotation pool",
		Args:  cobra.ExactArgs(2),
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
