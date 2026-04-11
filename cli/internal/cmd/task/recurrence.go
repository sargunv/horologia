package taskcmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
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
		Args:  cobra.ExactArgs(2),
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

	cmd.Flags().StringVar(&recurrenceType, "type", "", "Recurrence type")
	cmd.Flags().StringVar(&rule, "rule", "", "RRULE string")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newRecurrenceClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <task>",
		Short: "Clear task recurrence",
		Args:  cobra.ExactArgs(2),
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
