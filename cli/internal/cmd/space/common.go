package spacecmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/cli/internal/runtime"
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

func printSpace(app *runtime.App, space *apigen.Space) {
	app.Printf("Name:        %s\n", space.Name)
	app.Printf("Slug:        %s\n", space.Slug)
	app.Printf("Description: %s\n", space.Description)
	app.Printf("Created:     %s\n", space.CreatedAt.Format(timeFormat))
	app.Printf("Updated:     %s\n", space.UpdatedAt.Format(timeFormat))
}

func printSpaceList(app *runtime.App, spaces []apigen.Space) error {
	if len(spaces) == 0 {
		app.Printf("No spaces.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("SLUG\tNAME\tDESCRIPTION\n"))
	for _, space := range spaces {
		_, _ = w.Write([]byte(space.Slug + "\t" + space.Name + "\t" + space.Description + "\n"))
	}
	return w.Flush()
}

func printMemberList(app *runtime.App, members []apigen.SpaceMember) error {
	if len(members) == 0 {
		app.Printf("No members.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("USER ID\tNAME\tEMAIL\tROLE\tCREATED\n"))
	for _, member := range members {
		_, _ = w.Write([]byte(
			member.UserId + "\t" +
				member.UserName + "\t" +
				member.UserEmail + "\t" +
				string(member.Role) + "\t" +
				member.CreatedAt.Format(timeFormat) + "\n",
		))
	}
	return w.Flush()
}

func printTagList(app *runtime.App, tags []apigen.Tag) error {
	if len(tags) == 0 {
		app.Printf("No tags.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("NAME\tCREATED\n"))
	for _, tag := range tags {
		_, _ = w.Write([]byte(tag.Name + "\t" + tag.CreatedAt.Format(timeFormat) + "\n"))
	}
	return w.Flush()
}

func printTaskStatusList(app *runtime.App, items []apigen.TaskStatus) error {
	if len(items) == 0 {
		app.Printf("No task statuses.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("POSITION\tNAME\tCATEGORY\n"))
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", item.Position, item.Name, item.Category)
	}
	return w.Flush()
}

func printTaskEffortLevelList(app *runtime.App, items []apigen.TaskEffortLevel) error {
	if len(items) == 0 {
		app.Printf("No task effort levels.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("POSITION\tNAME\n"))
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%d\t%s\n", item.Position, item.Name)
	}
	return w.Flush()
}

func printTaskPriorityLevelList(app *runtime.App, items []apigen.TaskPriorityLevel) error {
	if len(items) == 0 {
		app.Printf("No task priority levels.\n")
		return nil
	}

	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("POSITION\tNAME\n"))
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%d\t%s\n", item.Position, item.Name)
	}
	return w.Flush()
}

func parseSpaceRole(raw string) (apigen.SpaceRole, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("role is required")
	}

	var role apigen.SpaceRole
	if err := role.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("invalid role %q (expected one of: admin, member, viewer)", raw)
	}
	return role, nil
}

func trimRequiredValues(raw []string, field string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("at least one %s is required", field)
	}

	return trimOptionalValues(raw, field)
}

func trimOptionalValues(raw []string, field string) ([]string, error) {
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
