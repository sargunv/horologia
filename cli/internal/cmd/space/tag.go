package spacecmd

import (
	"strings"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/cli/internal/cmd/support"
	"github.com/sargunv/tend/cli/internal/runtime"
)

func newTagCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("tag", "Manage tags in a space")
	cmd.AddCommand(
		newTagListCmd(flags),
		newTagCreateCmd(flags),
		newTagRenameCmd(flags),
		newTagDeleteCmd(flags),
	)
	return cmd
}

func newTagListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space>",
		Short: "List tags in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			resp, err := api.SpaceTagsList(cmd.Context(), apigen.SpaceTagsListParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(resp)
			}

			return printTagList(app, resp.Items)
		}),
	}
}

func newTagCreateCmd(flags *support.RootFlags) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "create <space>",
		Short: "Create a tag in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			tag, err := api.SpaceTagsCreate(cmd.Context(), &apigen.TagCreate{
				Name: strings.TrimSpace(name),
			}, apigen.SpaceTagsCreateParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(tag)
			}

			app.Printf("Created tag %s in %s\n", tag.Name, args[0])
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Tag name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTagRenameCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <space> <tag> <new-name>",
		Short: "Rename a tag",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			tag, err := api.SpaceTagsUpdate(cmd.Context(), &apigen.TagUpdate{
				Name: strings.TrimSpace(args[2]),
			}, apigen.SpaceTagsUpdateParams{
				SpaceSlug: args[0],
				TagName:   args[1],
			})
			if err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(tag)
			}

			app.Printf("Renamed tag %s to %s in %s\n", args[1], tag.Name, args[0])
			return nil
		}),
	}
}

func newTagDeleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <space> <tag>",
		Short: "Delete a tag",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}

			tagName := strings.TrimSpace(args[1])
			if err := api.SpaceTagsDelete(cmd.Context(), apigen.SpaceTagsDeleteParams{
				SpaceSlug: args[0],
				TagName:   tagName,
			}); err != nil {
				return runtime.NormalizeError(err)
			}

			if app.Config.JSON {
				return app.PrintJSON(map[string]any{
					"spaceSlug": args[0],
					"tagName":   tagName,
					"deleted":   true,
				})
			}

			app.Printf("Deleted tag %s from %s\n", tagName, args[0])
			return nil
		}),
	}
}
