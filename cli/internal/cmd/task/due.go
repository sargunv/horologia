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
		Long: `Set the due date and timezone for a task. Both --date and --timezone
are required. The date must use YYYY-MM-DD format, and the timezone
must be a valid IANA identifier such as America/New_York.`,
		Example: `  # Set a due date in US Eastern time
  tend task due set my-project SV-42 \
    --date 2026-05-01 --timezone America/New_York

  # Set a due date in UTC
  tend task due set my-project SV-42 \
    --date 2026-06-15 --timezone UTC`,
		Args: cobra.ExactArgs(2),
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

	cmd.Flags().StringVar(&date, "date", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "IANA timezone, e.g. America/New_York")
	_ = cmd.MarkFlagRequired("date")
	_ = cmd.MarkFlagRequired("timezone")
	return cmd
}

func newDueClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task due date",
		Long: `Remove the due date from a task. The task will no longer appear in
overdue filters or trigger overdue actions.`,
		Example: `  # Clear the due date
  tend task due clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
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
