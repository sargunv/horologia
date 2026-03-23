package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/api/gen"

	"github.com/sargunv/tend/cli/internal/output"
)

var spaceSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"slug":        map[string]any{"type": "string"},
		"name":        map[string]any{"type": "string"},
		"description": map[string]any{"type": "string"},
		"createdAt":   map[string]any{"type": "string", "format": "date-time"},
		"updatedAt":   map[string]any{"type": "string", "format": "date-time"},
	},
	"required": []string{"slug", "name", "description", "createdAt", "updatedAt"},
}

var spaceHeaders = []string{"Slug", "Name", "Description", "Created", "Updated"}

func spaceRow(s gen.Space) []string {
	return []string{
		s.Slug,
		s.Name,
		s.Description,
		s.CreatedAt.Format(time.DateOnly),
		s.UpdatedAt.Format(time.DateOnly),
	}
}

func spaceKV(s gen.Space) []output.KV {
	return []output.KV{
		{Key: "Slug", Value: s.Slug},
		{Key: "Name", Value: s.Name},
		{Key: "Description", Value: s.Description},
		{Key: "Created", Value: s.CreatedAt.Format(time.RFC3339)},
		{Key: "Updated", Value: s.UpdatedAt.Format(time.RFC3339)},
	}
}

func newSpacesCmd() *cobra.Command {
	spacesCmd := &cobra.Command{
		Use:   "spaces",
		Short: "Manage spaces",
		Long:  "Create, list, update, and delete spaces. Spaces are containers for tasks.",
	}

	spacesCmd.AddCommand(
		newSpacesListCmd(),
		newSpacesGetCmd(),
		newSpacesCreateCmd(),
		newSpacesUpdateCmd(),
		newSpacesDeleteCmd(),
	)

	return spacesCmd
}

func newSpacesListCmd() *cobra.Command {
	var cursor string
	var limit int32

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List spaces",
		Long:  "List all spaces you have access to. Owners see all spaces; other users see only spaces they are members of.",
		Example: `  tend spaces list
  tend spaces list --limit 10
  tend spaces list --cursor '<cursor from previous result>'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintList(app.Printer, output.ListView[gen.Space]{ItemSchema: spaceSchema})
			}

			params := gen.SpacesListParams{}
			if cmd.Flags().Changed("cursor") {
				params.Cursor = gen.NewOptString(cursor)
			}
			if cmd.Flags().Changed("limit") {
				params.Limit = gen.NewOptInt32(limit)
			}

			page, err := app.Client.SpacesList(cmd.Context(), params)
			if err != nil {
				return FormatAPIError(err)
			}

			nextCursor, _ := page.NextCursor.Get()

			return output.PrintList(app.Printer, output.ListView[gen.Space]{
				Items:      page.Items,
				NextCursor: nextCursor,
				ItemSchema: spaceSchema,
				Headers:    spaceHeaders,
				RowFunc:    spaceRow,
			})
		},
	}

	cmd.Flags().StringVar(&cursor, "cursor", "", "Resume from a cursor returned by a previous list call")
	cmd.Flags().Int32Var(&limit, "limit", 0, "Maximum number of results per page (server default if unset)")

	return cmd
}

func newSpacesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <slug>",
		Short:   "Get a space by slug",
		Long:    "Display full details for the space identified by the given slug.",
		Args:    cobra.ExactArgs(1),
		Example: "  tend spaces get engineering",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{Schema: spaceSchema})
			}

			space, err := app.Client.SpacesRead(cmd.Context(), gen.SpacesReadParams{
				SpaceSlug: args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{
				Value:  *space,
				Schema: spaceSchema,
				Rows:   spaceKV(*space),
			})
		},
	}
}

func newSpacesCreateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "create <slug>",
		Short: "Create a new space",
		Long:  "Create a new space. The slug argument is a unique identifier used in URLs and CLI commands.",
		Args:  cobra.ExactArgs(1),
		Example: `  tend spaces create engineering --name "Engineering"
  tend spaces create design --name "Design" --description "Design team tasks"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{Schema: spaceSchema})
			}

			req := &gen.SpaceCreate{
				Slug: args[0],
				Name: name,
			}
			if description != "" {
				req.Description = gen.NewOptString(description)
			}

			space, err := app.Client.SpacesCreate(cmd.Context(), req)
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{
				Value:  *space,
				Schema: spaceSchema,
				Rows:   spaceKV(*space),
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Space name (required)")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&description, "description", "", "Space description")

	return cmd
}

func newSpacesUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update a space",
		Long:  "Update one or more fields of an existing space. Only the flags you specify are changed.",
		Args:  cobra.ExactArgs(1),
		Example: `  tend spaces update engineering --name "Eng"
  tend spaces update engineering --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{Schema: spaceSchema})
			}

			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") {
				return fmt.Errorf("specify at least one of --name or --description")
			}

			req := &gen.SpaceUpdate{}
			if cmd.Flags().Changed("name") {
				req.Name = gen.NewOptString(name)
			}
			if cmd.Flags().Changed("description") {
				req.Description = gen.NewOptString(description)
			}

			space, err := app.Client.SpacesUpdate(cmd.Context(), req, gen.SpacesUpdateParams{
				SpaceSlug: args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			return output.PrintResource(app.Printer, output.ResourceView[gen.Space]{
				Value:  *space,
				Schema: spaceSchema,
				Rows:   spaceKV(*space),
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Space name")
	cmd.Flags().StringVar(&description, "description", "", "Space description")

	return cmd
}

func newSpacesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <slug>",
		Short:   "Delete a space",
		Long:    "Permanently delete a space and all of its tasks.",
		Args:    cobra.ExactArgs(1),
		Example: "  tend spaces delete engineering",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)

			if app.Printer.IsSchemaMode() {
				return output.ErrNoSchema
			}

			err := app.Client.SpacesDelete(cmd.Context(), gen.SpacesDeleteParams{
				SpaceSlug: args[0],
			})
			if err != nil {
				return FormatAPIError(err)
			}

			app.Printer.PrintDeletion("space", args[0])
			return nil
		},
	}
}
