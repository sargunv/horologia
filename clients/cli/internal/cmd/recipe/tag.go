package recipecmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newTagCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("tag", "Manage recipe tags")
	cmd.AddCommand(newTagSetCmd(flags), newTagAddCmd(flags), newTagRemoveCmd(flags), newTagClearCmd(flags))
	return cmd
}

func newTagSetCmd(flags *support.RootFlags) *cobra.Command {
	var tags []string
	cmd := &cobra.Command{
		Use:   "set <space> <recipe>",
		Short: "Replace recipe tags",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			values, err := trimRequiredStrings(tags, "tag")
			if err != nil {
				return err
			}
			recipe, err := updateRecipe(cmd.Context(), api, args[0], args[1], &apigen.RecipeUpdate{Tags: uniqueStrings(values)})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			return printMutationResult(app, recipe)
		}),
	}
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag name (repeatable)")
	return cmd
}

func newTagAddCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <space> <recipe> <tag>",
		Short: "Add a tag to a recipe",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			tag := strings.TrimSpace(args[2])
			if tag == "" {
				return errors.New("tag cannot be empty")
			}
			updated, err := updateRecipe(cmd.Context(), api, args[0], args[1], &apigen.RecipeUpdate{Tags: uniqueStrings(append(append([]string{}, recipe.Tags...), tag))})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			return printMutationResult(app, updated)
		}),
	}
}

func newTagRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <recipe> <tag>",
		Short: "Remove a tag from a recipe",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			updated, err := updateRecipe(cmd.Context(), api, args[0], args[1], &apigen.RecipeUpdate{Tags: withoutString(recipe.Tags, strings.TrimSpace(args[2]))})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			return printMutationResult(app, updated)
		}),
	}
}

func newTagClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <recipe>",
		Short: "Clear recipe tags",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := updateRecipe(cmd.Context(), api, args[0], args[1], &apigen.RecipeUpdate{Tags: []string{}})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			return printMutationResult(app, recipe)
		}),
	}
}

func printMutationResult(app *runtime.App, recipe *apigen.Recipe) error {
	if app.Config.JSON {
		return app.PrintJSON(recipe)
	}
	printRecipe(app, recipe)
	return nil
}
