package usercmd

import (
	"strings"
	"text/tabwriter"

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

func printUser(app *runtime.App, user *apigen.User) {
	app.Printf("Name:        %s\n", user.Name)
	app.Printf("Email:       %s\n", user.Email)
	app.Printf("ID:          %s\n", user.ID)
	app.Printf("Owner:       %t\n", user.IsOwner)
	app.Printf("HasPassword: %t\n", user.HasPassword)
	app.Printf("Created:     %s\n", user.CreatedAt.Format(timeFormat))
	app.Printf("Updated:     %s\n", user.UpdatedAt.Format(timeFormat))
}

func printUserList(app *runtime.App, users []apigen.User) error {
	if len(users) == 0 {
		app.Printf("No users.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("ID\tNAME\tEMAIL\tOWNER\tPASSWORD\n"))
	for _, user := range users {
		_, _ = w.Write([]byte(user.ID + "\t" + user.Name + "\t" + user.Email + "\t" + formatBool(user.IsOwner) + "\t" + formatBool(user.HasPassword) + "\n"))
	}
	return w.Flush()
}

func printTaskPage(app *runtime.App, page *apigen.TaskPage) {
	if len(page.Items) == 0 {
		app.Printf("No tasks.\n")
	} else {
		for _, task := range page.Items {
			app.Printf("%s [%s] %s (%s)\n", task.ID, task.Status, task.Title, task.SpaceSlug)
			if due, ok := task.Due.Get(); ok {
				app.Printf("  Due: %s %s\n", due.At.Format("2006-01-02"), due.Timezone)
			}
			if len(task.AssigneeIds) > 0 {
				app.Printf("  Assignees: %s\n", strings.Join(task.AssigneeIds, ", "))
			}
			if len(task.Tags) > 0 {
				app.Printf("  Tags: %s\n", strings.Join(task.Tags, ", "))
			}
		}
	}

	if next, ok := page.NextCursor.Get(); ok {
		app.Printf("Next cursor: %s\n", next)
	}
}

func printActivityPage(app *runtime.App, page *apigen.ActivityLogPage) {
	if len(page.Items) == 0 {
		app.Printf("No activity.\n")
	} else {
		for _, entry := range page.Items {
			app.Printf(
				"%s %s %s %s %s\n",
				entry.CreatedAt.Format(timeFormat),
				entry.EntityType,
				entry.EntityId,
				entry.Action,
				entry.SpaceSlug,
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

func formatBool(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
