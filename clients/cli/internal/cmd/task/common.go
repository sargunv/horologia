package taskcmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

const timeFormat = "2006-01-02 15:04:05Z07:00"

type pageFlags struct {
	cursor string
	limit  int32
}

func addPageFlags(cmd *cobra.Command, flags *pageFlags) {
	cmd.Flags().StringVar(&flags.cursor, "cursor", "", "Pagination cursor from a previous response")
	cmd.Flags().Int32Var(&flags.limit, "limit", 0, "Maximum number of items to return")
}

func setPageParams(cursor string, limit int32, cursorDst *apigen.OptString, limitDst *apigen.OptInt32) {
	if strings.TrimSpace(cursor) != "" {
		cursorDst.SetTo(strings.TrimSpace(cursor))
	}
	if limit > 0 {
		limitDst.SetTo(limit)
	}
}

func printTask(app *runtime.App, task *apigen.Task) {
	app.Printf("ID:             %s\n", task.ID)
	app.Printf("Space:          %s\n", task.SpaceSlug)
	app.Printf("Title:          %s\n", task.Title)
	app.Printf("Description:    %s\n", task.Description)
	app.Printf("Status:         %s\n", task.Status)
	app.Printf("Effort:         %s\n", formatNilString(task.Effort))
	app.Printf("Priority:       %s\n", formatNilString(task.Priority))
	app.Printf("RecurrenceType: %s\n", task.RecurrenceType)
	app.Printf("RecurrenceRule: %s\n", formatNilString(task.RecurrenceRule))
	app.Printf("Due:            %s\n", formatDue(task.Due))
	app.Printf("OverdueAction:  %s\n", formatOverdueActionRule(task.OverdueActionRule))
	app.Printf("Assignees:      %s\n", formatStringList(task.AssigneeIds))
	app.Printf("RotationPool:   %s\n", formatStringList(task.RotationPool))
	app.Printf("Tags:           %s\n", formatStringList(task.Tags))
	if len(task.Relations) == 0 {
		app.Printf("Relations:      none\n")
	} else {
		app.Printf("Relations:\n")
		for _, relation := range task.Relations {
			app.Printf("  %s %s (%s)\n", relation.Kind, relation.RelatedTaskId, relation.CreatedAt.Format(timeFormat))
		}
	}
	app.Printf("Created:        %s\n", task.CreatedAt.Format(timeFormat))
	app.Printf("Updated:        %s\n", task.UpdatedAt.Format(timeFormat))
	if lastCompleted, ok := task.LastCompletedAt.Get(); ok {
		app.Printf("LastCompleted:  %s\n", lastCompleted.Format(timeFormat))
	}
}

func printActivityPage(app *runtime.App, page *apigen.ActivityLogPage) {
	if len(page.Items) == 0 {
		app.Printf("No activity.\n")
	} else {
		for _, entry := range page.Items {
			app.Printf(
				"%s %s %s %s\n",
				entry.CreatedAt.Format(timeFormat),
				entry.EntityType,
				entry.EntityId,
				entry.Action,
			)
			if tokenName, ok := entry.TokenName.Get(); ok {
				app.Printf("  Token: %s\n", tokenName)
			}
			for _, detail := range entry.Details {
				from, hasFrom := detail.From.Get()
				to, hasTo := detail.To.Get()
				switch {
				case hasFrom && hasTo:
					app.Printf("  %s: %s -> %s\n", detail.Field, from, to)
				case hasTo:
					app.Printf("  %s: -> %s\n", detail.Field, to)
				case hasFrom:
					app.Printf("  %s: %s ->\n", detail.Field, from)
				default:
					app.Printf("  %s\n", detail.Field)
				}
			}
		}
	}

	if next, ok := page.NextCursor.Get(); ok {
		app.Printf("Next cursor: %s\n", next)
	}
}

func parseDueDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q; expected YYYY-MM-DD", raw)
	}
	return date, nil
}

func trimRequiredStrings(raw []string, field string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("at least one %s is required", field)
	}
	return trimOptionalStrings(raw, field)
}

func trimOptionalStrings(raw []string, field string) ([]string, error) {
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		value := strings.TrimSpace(item)
		if value == "" {
			return nil, fmt.Errorf("%s cannot be empty", field)
		}
		values = append(values, value)
	}
	return values, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func withoutString(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == target {
			continue
		}
		result = append(result, value)
	}
	return result
}

func parseRelationKind(raw string) (apigen.TaskRelationKind, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("relation kind is required")
	}

	var kind apigen.TaskRelationKind
	if err := kind.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("invalid relation kind %q", raw)
	}
	return kind, nil
}

func parseRecurrenceType(raw string) (apigen.TaskRecurrenceType, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("recurrence type is required")
	}

	var recurrenceType apigen.TaskRecurrenceType
	if err := recurrenceType.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("invalid recurrence type %q", raw)
	}
	return recurrenceType, nil
}

func recurrenceTypeRequiresRule(value apigen.TaskRecurrenceType) bool {
	switch value {
	case apigen.TaskRecurrenceTypeCompletionBased,
		apigen.TaskRecurrenceTypeFixedNonAccumulating,
		apigen.TaskRecurrenceTypeFixedAccumulating:
		return true
	default:
		return false
	}
}

func parseOverdueAction(raw string) (apigen.TaskOverdueAction, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("overdue action is required")
	}

	var action apigen.TaskOverdueAction
	if err := action.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("invalid overdue action %q", raw)
	}
	return action, nil
}

func readTask(ctx context.Context, api *apigen.Client, spaceSlug string, taskID string) (*apigen.Task, error) {
	return api.SpaceTasksRead(ctx, apigen.SpaceTasksReadParams{
		SpaceSlug: spaceSlug,
		TaskId:    taskID,
	})
}

func firstCompletionStatus(statuses []apigen.TaskStatus) (string, error) {
	for _, status := range statuses {
		if status.Category == apigen.TaskStatusCategoryCompletion {
			return status.Name, nil
		}
	}
	return "", fmt.Errorf("space has no completion status")
}

func formatNilString(value apigen.NilString) string {
	if s, ok := value.Get(); ok {
		return s
	}
	return "none"
}

func formatDue(value apigen.NilTaskDue) string {
	if due, ok := value.Get(); ok {
		return due.At.Format("2006-01-02") + " " + due.Timezone
	}
	return "none"
}

func formatOverdueActionRule(value apigen.NilTaskOverdueActionRule) string {
	rule, ok := value.Get()
	if !ok {
		return "none"
	}

	after := "immediate"
	if days, ok := rule.After.Get(); ok {
		after = fmt.Sprintf("%d day(s)", days)
	}

	if rule.Action == apigen.TaskOverdueActionSetStatus {
		status, _ := rule.Status.Get()
		return fmt.Sprintf("%s after %s", rule.Action, after) + " -> " + status
	}
	return fmt.Sprintf("%s after %s", rule.Action, after)
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func printTaskListTable(app *runtime.App, tasks []apigen.Task) error {
	if len(tasks) == 0 {
		app.Printf("No tasks.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("ID\tSTATUS\tTITLE\tSPACE\tDUE\n"))
	for _, task := range tasks {
		due := ""
		if value, ok := task.Due.Get(); ok {
			due = value.At.Format("2006-01-02")
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", task.ID, task.Status, task.Title, task.SpaceSlug, due)
	}
	return w.Flush()
}

func printTaskSearchResultTable(app *runtime.App, tasks []apigen.TaskSearchResult) error {
	if len(tasks) == 0 {
		app.Printf("No tasks.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("ID\tSTATUS\tTITLE\tSPACE\n"))
	for _, task := range tasks {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ID, task.Status, task.Title, task.SpaceSlug)
	}
	return w.Flush()
}
