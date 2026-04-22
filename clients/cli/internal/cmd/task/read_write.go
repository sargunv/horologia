package taskcmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newReadWriteCmds(flags *support.RootFlags) []*cobra.Command {
	return []*cobra.Command{
		newListCmd(flags),
		newMineCmd(flags),
		newShowCmd(flags),
		newCreateCmd(flags),
		newUpdateCmd(flags),
		newCompleteCmd(flags),
		newDeleteCmd(flags),
		newActivityCmd(flags),
	}
}

func newListCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "list <space>",
		Short: "List tasks in a space",
		Long: `List all tasks in the given space. Results are paginated; use --cursor
with the value from a previous response to fetch the next page.`,
		Example: `  # See all tasks in a space
  horo task list my-project

  # Limit to the first 10 results
  horo task list my-project --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			params := apigen.SpaceTasksListParams{SpaceSlug: args[0]}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)

			resp, err := api.SpaceTasksList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskListTable(app, resp.Items)
		}),
	}

	addPageFlags(cmd, &page)
	return cmd
}

func newMineCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "mine",
		Short: "List tasks assigned to the current user",
		Long: `List tasks assigned to the authenticated user across all spaces.
Results are paginated; use --cursor with the value from a previous
response to fetch the next page.`,
		Example: `  # See what's on your plate
  horo task mine

  # Limit to the first 5 results
  horo task mine --limit 5`,
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			me, err := api.UsersMe(cmd.Context())
			if err != nil {
				return runtime.NormalizeError(err)
			}

			params := apigen.UserTasksListParams{UserId: me.ID}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)

			resp, err := api.UserTasksList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTaskListTable(app, resp.Items)
		}),
	}

	addPageFlags(cmd, &page)
	return cmd
}

func newShowCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <space> <task>",
		Short: "Show a task",
		Long: `Display full details for a single task. The <space> argument is
the space slug and <task> is the task ID.`,
		Example: `  # Inspect a single task by ID
  horo task show my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			task, err := api.SpaceTasksRead(cmd.Context(), apigen.SpaceTasksReadParams{
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

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var description string
	var status string
	var effort string
	var priority string

	cmd := &cobra.Command{
		Use:   "create <space>",
		Short: "Create a task",
		Long: `Create a new task in the given space. The --title flag is required;
all other fields are optional and use the space's defaults when omitted.`,
		Example: `  # Create a task with just a title
  horo task create my-project --title "Fix login bug"

  # Create a task with all fields
  horo task create my-project --title "Fix login bug" \
    --status "In Progress" --priority High --effort Medium`,
		Args: cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskCreate{Title: strings.TrimSpace(title)}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(description)
			}
			if cmd.Flags().Changed("status") {
				req.Status.SetTo(strings.TrimSpace(status))
			}
			if cmd.Flags().Changed("effort") {
				req.Effort.SetTo(strings.TrimSpace(effort))
			}
			if cmd.Flags().Changed("priority") {
				req.Priority.SetTo(strings.TrimSpace(priority))
			}

			task, err := api.SpaceTasksCreate(cmd.Context(), req, apigen.SpaceTasksCreateParams{SpaceSlug: args[0]})
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

	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringVar(&status, "status", "", "Task status")
	cmd.Flags().StringVar(&effort, "effort", "", "Task effort level")
	cmd.Flags().StringVar(&priority, "priority", "", "Task priority level")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var description string
	var status string
	var effort string
	var clearEffort bool
	var priority string
	var clearPriority bool

	cmd := &cobra.Command{
		Use:   "update <space> <task>",
		Short: "Update a task",
		Long: `Update fields on an existing task. Only the fields you specify will
change; all other fields remain untouched. Use --clear-effort or
--clear-priority to remove an optional field value.`,
		Example: `  # Change the title
  horo task update my-project SV-42 --title "New title"

  # Set priority and clear effort
  horo task update my-project SV-42 --priority High --clear-effort`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			changed := false

			if cmd.Flags().Changed("title") {
				req.Title.SetTo(strings.TrimSpace(title))
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(description)
				changed = true
			}
			if cmd.Flags().Changed("status") {
				req.Status.SetTo(strings.TrimSpace(status))
				changed = true
			}
			if cmd.Flags().Changed("effort") {
				req.Effort.SetTo(strings.TrimSpace(effort))
				changed = true
			}
			if cmd.Flags().Changed("clear-effort") && clearEffort {
				req.Effort.SetToNull()
				changed = true
			}
			if cmd.Flags().Changed("priority") {
				req.Priority.SetTo(strings.TrimSpace(priority))
				changed = true
			}
			if cmd.Flags().Changed("clear-priority") && clearPriority {
				req.Priority.SetToNull()
				changed = true
			}

			if !changed {
				return errors.New("at least one field flag is required")
			}
			if req.Effort.IsSet() && req.Effort.IsNull() && cmd.Flags().Changed("effort") {
				return errors.New("effort and clear-effort cannot be used together")
			}
			if req.Priority.IsSet() && req.Priority.IsNull() && cmd.Flags().Changed("priority") {
				return errors.New("priority and clear-priority cannot be used together")
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

	cmd.Flags().StringVar(&title, "title", "", "Updated task title")
	cmd.Flags().StringVar(&description, "description", "", "Updated task description")
	cmd.Flags().StringVar(&status, "status", "", "Updated task status")
	cmd.Flags().StringVar(&effort, "effort", "", "Updated task effort level")
	cmd.Flags().BoolVar(&clearEffort, "clear-effort", false, "Clear the task effort level")
	cmd.Flags().StringVar(&priority, "priority", "", "Updated task priority level")
	cmd.Flags().BoolVar(&clearPriority, "clear-priority", false, "Clear the task priority level")
	return cmd
}

func newCompleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "complete <space> <task>",
		Short: "Mark a task complete",
		Long:  `Mark a task complete by setting it to the space's first completion status.`,
		Example: `  # Finish a task
  horo task complete my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			statuses, err := api.SpaceTaskStatusesList(cmd.Context(), apigen.SpaceTaskStatusesListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			completionStatus, err := firstCompletionStatus(statuses.Items)
			if err != nil {
				return err
			}

			req := &apigen.TaskUpdate{}
			req.Status.SetTo(completionStatus)
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

func newDeleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <space> <task>",
		Short: "Delete a task",
		Long: `Permanently delete a task and all its associated data. This cannot
be undone.`,
		Example: `  # Permanently remove a task
  horo task delete my-project SV-42`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			if err := api.SpaceTasksDelete(cmd.Context(), apigen.SpaceTasksDeleteParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"spaceSlug": args[0],
					"taskId":    args[1],
					"deleted":   true,
				})
			}

			app.Printf("Deleted task %s from %s\n", args[1], args[0])
			return nil
		}),
	}
}

func newActivityCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}

	cmd := &cobra.Command{
		Use:   "activity <space> <task>",
		Short: "Show activity for a task",
		Long: `Show the activity log for a task, including status changes, field
updates, and comments. Results are paginated; use --cursor with the
value from a previous response to fetch the next page.`,
		Example: `  # Show activity for a task
  horo task activity my-project SV-42

  # Show the last 5 activity entries
  horo task activity my-project SV-42 --limit 5`,
		Args: cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			params := apigen.SpaceTaskActivityListParams{
				SpaceSlug: args[0],
				TaskId:    args[1],
			}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)

			resp, err := api.SpaceTaskActivityList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			printActivityPage(app, resp)
			return nil
		}),
	}

	addPageFlags(cmd, &page)
	return cmd
}
