package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/api/gen"

	"github.com/sargunv/tend/cli/internal/output"
)

var taskSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"id":          map[string]any{"type": "string"},
		"title":       map[string]any{"type": "string"},
		"description": map[string]any{"type": "string"},
		"status":      map[string]any{"type": "string"},
		"effort":      map[string]any{"type": []string{"string", "null"}},
		"priority":    map[string]any{"type": []string{"string", "null"}},
		"assigneeIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"tags":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"relations": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind":      map[string]any{"type": "string"},
				"taskId":    map[string]any{"type": "string"},
				"createdAt": map[string]any{"type": "string", "format": "date-time"},
			},
			"required": []string{"kind", "taskId", "createdAt"},
		}},
		"dueDate":   map[string]any{"type": []string{"string", "null"}, "format": "date"},
		"createdAt": map[string]any{"type": "string", "format": "date-time"},
		"updatedAt": map[string]any{"type": "string", "format": "date-time"},
	},
	"required": []string{"id", "title", "description", "status", "effort", "priority", "assigneeIds", "tags", "relations", "dueDate", "createdAt", "updatedAt"},
}

var taskHeaders = []string{"ID", "Title", "Status", "Effort", "Priority", "Due", "Assignees", "Created"}

func formatDue(t gen.Task, fallback string) string {
	if d, ok := t.DueDate.Get(); ok {
		return d.Format(time.DateOnly)
	}
	return fallback
}

// formatAssignees joins assignee IDs for display, truncating after maxShow.
func formatAssignees(t gen.Task, fallback string, maxShow int) string {
	if len(t.AssigneeIds) == 0 {
		return fallback
	}
	if maxShow > 0 && len(t.AssigneeIds) > maxShow {
		return strings.Join(t.AssigneeIds[:maxShow], ", ") +
			fmt.Sprintf(" +%d more", len(t.AssigneeIds)-maxShow)
	}
	return strings.Join(t.AssigneeIds, ", ")
}

func parseDueDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("--due requires a date value in YYYY-MM-DD format")
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid due date %q: expected YYYY-MM-DD format", s)
	}
	return d, nil
}

func taskRow(t gen.Task) []string {
	return []string{
		t.ID,
		t.Title,
		t.Status,
		formatNilString(t.Effort, "-"),
		formatNilString(t.Priority, "-"),
		formatDue(t, "-"),
		formatAssignees(t, "-", 3),
		t.CreatedAt.Format(time.DateOnly),
	}
}

func formatNilString(ns gen.NilString, fallback string) string {
	if v, ok := ns.Get(); ok {
		return v
	}
	return fallback
}

func taskKV(t gen.Task) []output.KV {
	return []output.KV{
		{Key: "ID", Value: t.ID},
		{Key: "Title", Value: t.Title},
		{Key: "Description", Value: t.Description},
		{Key: "Status", Value: t.Status},
		{Key: "Effort", Value: formatNilString(t.Effort, "-")},
		{Key: "Priority", Value: formatNilString(t.Priority, "-")},
		{Key: "Due Date", Value: formatDue(t, "-")},
		{Key: "Assignees", Value: formatAssignees(t, "-", 0)},
		{Key: "Created", Value: t.CreatedAt.Format(time.RFC3339)},
		{Key: "Updated", Value: t.UpdatedAt.Format(time.RFC3339)},
	}
}

func newTasksCmd() *cobra.Command {
	tasksCmd := &cobra.Command{
		Use:   "tasks",
		Short: "Manage tasks",
		Long:  "Create, list, update, and delete tasks within a space.",
	}

	tasksCmd.AddCommand(
		newTasksListCmd(),
		newTasksGetCmd(),
		newTasksCreateCmd(),
		newTasksUpdateCmd(),
		newTasksDeleteCmd(),
	)

	return tasksCmd
}

// addSpaceFlag registers a required --space flag on the given command.
func addSpaceFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "space", "", "Space slug (required)")
	_ = cmd.MarkFlagRequired("space")
}

func newTasksListCmd() *cobra.Command {
	var space, cursor string
	var limit int32

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a space",
		Long:  "List all tasks in the given space. Use --cursor to page through results.",
		Example: `  tend tasks list --space engineering
  tend tasks list --space engineering --limit 10
  tend tasks list --space engineering --cursor '<cursor from previous result>'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintList(app.Printer, output.ListView[gen.Task]{ItemSchema: taskSchema})
			}

			params := gen.SpaceTasksListParams{
				SpaceSlug: space,
			}
			if cmd.Flags().Changed("cursor") {
				params.Cursor = gen.NewOptString(cursor)
			}
			if cmd.Flags().Changed("limit") {
				params.Limit = gen.NewOptInt32(limit)
			}

			page, err := app.Client.SpaceTasksList(cmd.Context(), params)
			if err != nil {
				return FormatAPIError(err)
			}

			nextCursor, _ := page.NextCursor.Get()

			return output.PrintList(app.Printer, output.ListView[gen.Task]{
				Items:      page.Items,
				NextCursor: nextCursor,
				ItemSchema: taskSchema,
				Headers:    taskHeaders,
				RowFunc:    taskRow,
			})
		},
	}

	addSpaceFlag(cmd, &space)
	cmd.Flags().StringVar(&cursor, "cursor", "", "Resume from a cursor returned by a previous list call")
	cmd.Flags().Int32Var(&limit, "limit", 0, "Maximum number of results per page (server default if unset)")

	return cmd
}

func newTasksGetCmd() *cobra.Command {
	var space string

	cmd := &cobra.Command{
		Use:     "get <task-id>",
		Short:   "Get a task by ID",
		Long:    "Display full details for the task identified by the given ID (e.g. T42).",
		Args:    cobra.ExactArgs(1),
		Example: "  tend tasks get --space engineering T42",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{Schema: taskSchema})
			}

			task, err := app.Client.SpaceTasksRead(cmd.Context(), gen.SpaceTasksReadParams{
				SpaceSlug: space,
				TaskId:    args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{
				Value:  *task,
				Schema: taskSchema,
				Rows:   taskKV(*task),
			})
		},
	}

	addSpaceFlag(cmd, &space)

	return cmd
}

func newTasksCreateCmd() *cobra.Command {
	var space, title, description, statusName, effortName, priorityName string
	var dueDate string
	var assignees []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		Long: `Create a new task in the given space. If --status is omitted, the server
assigns the default initial status. Status names are defined per space.`,
		Example: `  tend tasks create --space engineering --title "Fix login bug"
  tend tasks create --space engineering --title "Deploy v2" --due 2026-04-01 --assignee U1 --assignee U2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{Schema: taskSchema})
			}

			req := &gen.TaskCreate{
				Title: title,
			}
			if description != "" {
				req.Description = gen.NewOptString(description)
			}
			if statusName != "" {
				req.Status = gen.NewOptString(statusName)
			}
			if cmd.Flags().Changed("effort") {
				req.Effort = gen.NewOptString(effortName)
			}
			if cmd.Flags().Changed("priority") {
				req.Priority = gen.NewOptString(priorityName)
			}
			if cmd.Flags().Changed("due") {
				d, err := parseDueDate(dueDate)
				if err != nil {
					return err
				}
				req.DueDate = gen.NewOptNilDate(d)
			}
			if cmd.Flags().Changed("assignee") {
				req.AssigneeIds = assignees
			}

			task, err := app.Client.SpaceTasksCreate(cmd.Context(), req, gen.SpaceTasksCreateParams{
				SpaceSlug: space,
			})
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{
				Value:  *task,
				Schema: taskSchema,
				Rows:   taskKV(*task),
			})
		},
	}

	addSpaceFlag(cmd, &space)
	cmd.Flags().StringVar(&title, "title", "", "Task title (required)")
	_ = cmd.MarkFlagRequired("title")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringVar(&statusName, "status", "", "Status name (as defined in the space)")
	cmd.Flags().StringVar(&effortName, "effort", "", "Effort level (as defined in the space)")
	cmd.Flags().StringVar(&priorityName, "priority", "", "Priority level (as defined in the space)")
	cmd.Flags().StringVar(&dueDate, "due", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().StringArrayVar(&assignees, "assignee", nil, "Assignee user ID (can be repeated)")

	return cmd
}

func newTasksUpdateCmd() *cobra.Command {
	var space, title, description, statusName, effortName, priorityName string
	var dueDate string
	var assignees []string
	var clearDue, clearAssignees, clearEffort, clearPriority bool

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update a task",
		Long: `Update one or more fields of an existing task. Only the flags you specify
are changed. Use --clear-due to remove a due date, and --clear-assignees
to remove all assignees.`,
		Args: cobra.ExactArgs(1),
		Example: `  tend tasks update --space engineering T42 --title "New title"
  tend tasks update --space engineering T42 --due 2026-05-01
  tend tasks update --space engineering T42 --clear-due
  tend tasks update --space engineering T42 --assignee U1 --assignee U2
  tend tasks update --space engineering T42 --clear-assignees`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{Schema: taskSchema})
			}

			hasChanges := cmd.Flags().Changed("title") || cmd.Flags().Changed("description") ||
				cmd.Flags().Changed("status") || cmd.Flags().Changed("effort") ||
				cmd.Flags().Changed("clear-effort") || cmd.Flags().Changed("priority") ||
				cmd.Flags().Changed("clear-priority") || cmd.Flags().Changed("due") ||
				cmd.Flags().Changed("clear-due") || cmd.Flags().Changed("assignee") ||
				cmd.Flags().Changed("clear-assignees")
			if !hasChanges {
				return fmt.Errorf("specify at least one of --title, --description, --status, --effort, --clear-effort, --priority, --clear-priority, --due, --clear-due, --assignee, or --clear-assignees")
			}

			req := &gen.TaskUpdate{}
			if cmd.Flags().Changed("title") {
				req.Title = gen.NewOptString(title)
			}
			if cmd.Flags().Changed("description") {
				req.Description = gen.NewOptString(description)
			}
			if cmd.Flags().Changed("status") {
				req.Status = gen.NewOptString(statusName)
			}
			if clearEffort {
				req.Effort.SetToNull()
			} else if cmd.Flags().Changed("effort") {
				req.Effort = gen.NewOptNilString(effortName)
			}
			if clearPriority {
				req.Priority.SetToNull()
			} else if cmd.Flags().Changed("priority") {
				req.Priority = gen.NewOptNilString(priorityName)
			}
			if clearDue {
				req.DueDate.SetToNull()
			} else if cmd.Flags().Changed("due") {
				d, err := parseDueDate(dueDate)
				if err != nil {
					return err
				}
				req.DueDate = gen.NewOptNilDate(d)
			}
			if clearAssignees {
				req.AssigneeIds = []string{}
			} else if cmd.Flags().Changed("assignee") {
				req.AssigneeIds = assignees
			}

			task, err := app.Client.SpaceTasksUpdate(cmd.Context(), req, gen.SpaceTasksUpdateParams{
				SpaceSlug: space,
				TaskId:    args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Task]{
				Value:  *task,
				Schema: taskSchema,
				Rows:   taskKV(*task),
			})
		},
	}

	addSpaceFlag(cmd, &space)
	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringVar(&statusName, "status", "", "Status name (as defined in the space)")
	cmd.Flags().StringVar(&effortName, "effort", "", "Effort level (as defined in the space)")
	cmd.Flags().BoolVar(&clearEffort, "clear-effort", false, "Clear the effort level")
	cmd.Flags().StringVar(&priorityName, "priority", "", "Priority level (as defined in the space)")
	cmd.Flags().BoolVar(&clearPriority, "clear-priority", false, "Clear the priority level")
	cmd.Flags().StringVar(&dueDate, "due", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear the due date")
	cmd.Flags().StringArrayVar(&assignees, "assignee", nil, "Assignee user ID (can be repeated)")
	cmd.Flags().BoolVar(&clearAssignees, "clear-assignees", false, "Remove all assignees")
	cmd.MarkFlagsMutuallyExclusive("effort", "clear-effort")
	cmd.MarkFlagsMutuallyExclusive("priority", "clear-priority")
	cmd.MarkFlagsMutuallyExclusive("due", "clear-due")
	cmd.MarkFlagsMutuallyExclusive("assignee", "clear-assignees")

	return cmd
}

func newTasksDeleteCmd() *cobra.Command {
	var space string

	cmd := &cobra.Command{
		Use:     "delete <task-id>",
		Short:   "Delete a task",
		Long:    "Permanently delete a task.",
		Args:    cobra.ExactArgs(1),
		Example: "  tend tasks delete --space engineering T42",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.ErrNoSchema
			}

			err := app.Client.SpaceTasksDelete(cmd.Context(), gen.SpaceTasksDeleteParams{
				SpaceSlug: space,
				TaskId:    args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			app.Printer.PrintDeletion("task", args[0])
			return nil
		},
	}

	addSpaceFlag(cmd, &space)

	return cmd
}
