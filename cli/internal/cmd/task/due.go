package taskcmd

import (
	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newDueCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("due", "Manage task due dates")
	cmd.AddCommand(
		newDueSetCmd(flags),
		newDueClearCmd(flags),
	)
	return cmd
}

func newDueSetCmd(flags *support.RootFlags) *cobra.Command {
	var date string
	var timezone string

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Set a task due date",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			dueDate, err := parseDueDate(date)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			req.Due.SetTo(apigen.TaskDue{
				At:       dueDate,
				Timezone: timezone,
			})
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

	cmd.Flags().StringVar(&date, "date", "", "Due date in YYYY-MM-DD format")
	cmd.Flags().StringVar(&timezone, "timezone", "", "IANA timezone for the due date")
	_ = cmd.MarkFlagRequired("date")
	_ = cmd.MarkFlagRequired("timezone")
	return cmd
}

func newDueClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task due date",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			req.Due.SetToNull()
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
