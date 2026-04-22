package taskcmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newOverdueActionCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("overdue-action", "Manage task overdue actions")
	cmd.AddCommand(
		newOverdueActionSetCmd(flags),
		newOverdueActionClearCmd(flags),
	)
	return cmd
}

func newOverdueActionSetCmd(flags *support.RootFlags) *cobra.Command {
	var action string
	var status string
	var afterDays int32
	var immediate bool

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Set a task overdue action",
		Long: `Set the action taken when a task becomes overdue.

Valid --action values: advance_recurrence (advance to the next
recurrence), set_status (change to the status given by --status),
clear_due_date (remove the due date). The action fires immediately
unless --after-days sets a grace period.`,
		Example: `  # Advance recurrence immediately when overdue
  horo task overdue-action set my-project SV-42 \
    --action advance_recurrence

  # Mark a task done three days after it becomes overdue
  horo task overdue-action set my-project SV-42 \
    --action set_status --status Done --after-days 3`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			parsedAction, err := parseOverdueAction(action)
			if err != nil {
				return err
			}
			trimmedStatus := strings.TrimSpace(status)

			if parsedAction == apigen.TaskOverdueActionSetStatus && trimmedStatus == "" {
				return errors.New("status is required when action is set_status")
			}
			if parsedAction != apigen.TaskOverdueActionSetStatus && trimmedStatus != "" {
				return errors.New("status can only be used when action is set_status")
			}
			if cmd.Flags().Changed("after-days") && immediate {
				return errors.New("after-days and immediate cannot be used together")
			}
			if afterDays < 0 {
				return errors.New("after-days must be zero or greater")
			}

			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			rule := apigen.TaskOverdueActionRule{Action: parsedAction}
			if cmd.Flags().Changed("after-days") && !immediate {
				rule.After.SetTo(afterDays)
			} else {
				rule.After.SetToNull()
			}
			if parsedAction == apigen.TaskOverdueActionSetStatus {
				rule.Status.SetTo(trimmedStatus)
			}

			req := &apigen.TaskUpdate{}
			req.OverdueActionRule.SetTo(rule)
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

	cmd.Flags().StringVar(&action, "action", "", "Overdue action: advance_recurrence, set_status, clear_due_date")
	cmd.Flags().StringVar(&status, "status", "", "Target status name (required when --action is set_status)")
	cmd.Flags().Int32Var(&afterDays, "after-days", 0, "Grace period in days before the action fires")
	cmd.Flags().BoolVar(&immediate, "immediate", false, "Fire as soon as the task becomes overdue")
	_ = cmd.MarkFlagRequired("action")
	return cmd
}

func newOverdueActionClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task overdue action",
		Long: `Remove the overdue action from a task. The task will take no automatic
action when it becomes overdue.`,
		Example: `  # Remove the overdue action
  horo task overdue-action clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			req.OverdueActionRule.SetToNull()
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
