package cmd

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/api/gen"

	"github.com/sargunv/tend/cli/internal/output"
)

var userSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"id":        map[string]any{"type": "string"},
		"email":     map[string]any{"type": "string"},
		"name":      map[string]any{"type": "string"},
		"isOwner":   map[string]any{"type": "boolean"},
		"createdAt": map[string]any{"type": "string", "format": "date-time"},
		"updatedAt": map[string]any{"type": "string", "format": "date-time"},
	},
	"required": []string{"id", "email", "name", "isOwner", "createdAt", "updatedAt"},
}

func userKV(u *gen.User) []output.KV {
	return []output.KV{
		{Key: "ID", Value: u.ID},
		{Key: "Email", Value: u.Email},
		{Key: "Name", Value: u.Name},
		{Key: "Owner", Value: strconv.FormatBool(u.IsOwner)},
		{Key: "Created", Value: u.CreatedAt.Format(time.RFC3339)},
		{Key: "Updated", Value: u.UpdatedAt.Format(time.RFC3339)},
	}
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated user",
		Long:  "Show the name, email, and account details for the currently authenticated user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[*gen.User]{Schema: userSchema})
			}

			user, err := app.Client.UsersMe(cmd.Context())
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[*gen.User]{
				Value:  user,
				Schema: userSchema,
				Rows:   userKV(user),
			})
		},
	}
}
