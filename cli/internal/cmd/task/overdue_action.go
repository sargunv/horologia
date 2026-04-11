package taskcmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
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
		Args:  cobra.ExactArgs(2),
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

	cmd.Flags().StringVar(&action, "action", "", "Overdue action")
	cmd.Flags().StringVar(&status, "status", "", "Status to set when action is set_status")
	cmd.Flags().Int32Var(&afterDays, "after-days", 0, "Grace period in days before the action fires")
	cmd.Flags().BoolVar(&immediate, "immediate", false, "Fire immediately when the task becomes overdue")
	_ = cmd.MarkFlagRequired("action")
	return cmd
}

func newOverdueActionClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear a task overdue action",
		Args:  cobra.ExactArgs(2),
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
