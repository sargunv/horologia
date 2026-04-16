package taskcmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/cmd/support"
	"github.com/sargunv/horologia/cli/internal/runtime"
)

func newRecurrenceCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("recurrence", "Manage task recurrence settings")
	cmd.AddCommand(
		newRecurrenceSetCmd(flags),
		newRecurrenceClearCmd(flags),
	)
	return cmd
}

func newRecurrenceSetCmd(flags *support.RootFlags) *cobra.Command {
	var recurrenceType string
	var rule string

	cmd := &cobra.Command{
		Use:   "set <space> <task>",
		Short: "Set task recurrence",
		Long: `Set the recurrence schedule for a task. The --type flag selects
the strategy. Types that require an --rule (an RFC 5545 RRULE string):
completion_based, fixed_non_accumulating, fixed_accumulating. Types
that do not accept a rule: one_off, on_dependency.`,
		Example: `  # Recur weekly on completion
  horo task recurrence set my-project SV-42 \
    --type completion_based --rule "FREQ=WEEKLY;BYDAY=MO"

  # Recur on a fixed monthly schedule
  horo task recurrence set my-project SV-42 \
    --type fixed_non_accumulating --rule "FREQ=MONTHLY;BYMONTHDAY=1"`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			parsedType, err := parseRecurrenceType(recurrenceType)
			if err != nil {
				return err
			}

			trimmedRule := strings.TrimSpace(rule)
			if recurrenceTypeRequiresRule(parsedType) && trimmedRule == "" {
				return errors.New("rule is required for the selected recurrence type")
			}
			if !recurrenceTypeRequiresRule(parsedType) && trimmedRule != "" {
				return errors.New("rule cannot be used with the selected recurrence type")
			}

			req := &apigen.TaskUpdate{}
			req.RecurrenceType.SetTo(parsedType)
			if recurrenceTypeRequiresRule(parsedType) {
				req.RecurrenceRule.SetTo(trimmedRule)
			}

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

	cmd.Flags().StringVar(&recurrenceType, "type", "", "Recurrence type: one_off, completion_based, fixed_non_accumulating, fixed_accumulating, on_dependency")
	cmd.Flags().StringVar(&rule, "rule", "", "RFC 5545 RRULE string (required for completion_based, fixed_non_accumulating, fixed_accumulating)")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newRecurrenceClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear task recurrence",
		Long: `Remove the recurrence schedule from a task, resetting it to one_off.
The task keeps its current status and due date.`,
		Example: `  # Remove recurrence from a task
  horo task recurrence clear my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			req.RecurrenceType.SetTo(apigen.TaskRecurrenceTypeOneOff)
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
