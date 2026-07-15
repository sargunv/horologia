package recipecmd

import (
	"errors"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newListCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}
	cmd := &cobra.Command{
		Use:   "list <space>",
		Short: "List recipes in a space",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			params := apigen.SpaceRecipesListParams{SpaceSlug: args[0]}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)
			resp, err := api.SpaceRecipesList(cmd.Context(), params)
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(resp)
			}
			return printRecipeList(app, resp.Items)
		}),
	}
	addPageFlags(cmd, &page)
	return cmd
}

func newShowCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <space> <recipe>",
		Short: "Show a recipe",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(recipe)
			}
			printRecipe(app, recipe)
			return nil
		}),
	}
}

func newCreateCmd(flags *support.RootFlags) *cobra.Command {
	var name string
	var description string
	var recipeYield string
	var prep string
	var cook string
	var tags []string

	cmd := &cobra.Command{
		Use:   "create <space>",
		Short: "Create a recipe",
		Args:  cobra.ExactArgs(1),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			trimmedName := strings.TrimSpace(name)
			if trimmedName == "" {
				return errors.New("recipe name cannot be empty")
			}
			req := &apigen.RecipeCreate{Name: trimmedName}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(description)
			}
			if cmd.Flags().Changed("yield") {
				value, err := parseYield(recipeYield)
				if err != nil {
					return err
				}
				req.Yield.SetTo(value)
			}
			if cmd.Flags().Changed("prep") {
				value, err := parseDuration(prep)
				if err != nil {
					return err
				}
				req.PrepMinutes.SetTo(value)
			}
			if cmd.Flags().Changed("cook") {
				value, err := parseDuration(cook)
				if err != nil {
					return err
				}
				req.CookMinutes.SetTo(value)
			}
			if cmd.Flags().Changed("tag") {
				values, err := trimRequiredStrings(tags, "tag")
				if err != nil {
					return err
				}
				req.Tags = uniqueStrings(values)
			}

			recipe, err := api.SpaceRecipesCreate(cmd.Context(), req, apigen.SpaceRecipesCreateParams{SpaceSlug: args[0]})
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(recipe)
			}
			printRecipe(app, recipe)
			return nil
		}),
	}
	cmd.Flags().StringVar(&name, "name", "", "Recipe name")
	cmd.Flags().StringVar(&description, "description", "", "Recipe description")
	cmd.Flags().StringVar(&recipeYield, "yield", "", "Recipe yield, for example '4 servings'")
	cmd.Flags().StringVar(&prep, "prep", "", "Preparation duration, for example '20m' or '1h 15m'")
	cmd.Flags().StringVar(&cook, "cook", "", "Cooking duration, for example '45m'")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag name (repeatable)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var name string
	var description string
	var recipeYield string
	var clearYield bool
	var prep string
	var clearPrep bool
	var cook string
	var clearCook bool

	cmd := &cobra.Command{
		Use:   "update <space> <recipe>",
		Short: "Update a recipe",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("yield") && clearYield {
				return errors.New("yield and clear-yield cannot be used together")
			}
			if cmd.Flags().Changed("prep") && clearPrep {
				return errors.New("prep and clear-prep cannot be used together")
			}
			if cmd.Flags().Changed("cook") && clearCook {
				return errors.New("cook and clear-cook cannot be used together")
			}

			req := &apigen.RecipeUpdate{}
			changed := false
			if cmd.Flags().Changed("name") {
				value := strings.TrimSpace(name)
				if value == "" {
					return errors.New("recipe name cannot be empty")
				}
				req.Name.SetTo(value)
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description.SetTo(description)
				changed = true
			}
			if cmd.Flags().Changed("yield") {
				value, err := parseYield(recipeYield)
				if err != nil {
					return err
				}
				req.Yield.SetTo(value)
				changed = true
			} else if clearYield {
				req.Yield.SetToNull()
				changed = true
			}
			if cmd.Flags().Changed("prep") {
				value, err := parseDuration(prep)
				if err != nil {
					return err
				}
				req.PrepMinutes.SetTo(value)
				changed = true
			} else if clearPrep {
				req.PrepMinutes.SetToNull()
				changed = true
			}
			if cmd.Flags().Changed("cook") {
				value, err := parseDuration(cook)
				if err != nil {
					return err
				}
				req.CookMinutes.SetTo(value)
				changed = true
			} else if clearCook {
				req.CookMinutes.SetToNull()
				changed = true
			}
			if !changed {
				return errors.New("at least one field flag is required")
			}

			recipe, err := updateRecipe(cmd.Context(), api, args[0], args[1], req)
			if err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(recipe)
			}
			printRecipe(app, recipe)
			return nil
		}),
	}
	cmd.Flags().StringVar(&name, "name", "", "Updated recipe name")
	cmd.Flags().StringVar(&description, "description", "", "Updated recipe description")
	cmd.Flags().StringVar(&recipeYield, "yield", "", "Updated recipe yield")
	cmd.Flags().BoolVar(&clearYield, "clear-yield", false, "Clear the recipe yield")
	cmd.Flags().StringVar(&prep, "prep", "", "Updated preparation duration")
	cmd.Flags().BoolVar(&clearPrep, "clear-prep", false, "Clear the preparation duration")
	cmd.Flags().StringVar(&cook, "cook", "", "Updated cooking duration")
	cmd.Flags().BoolVar(&clearCook, "clear-cook", false, "Clear the cooking duration")
	return cmd
}

func newDeleteCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <space> <recipe>",
		Short: "Delete a recipe",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			if err := api.SpaceRecipesDelete(cmd.Context(), apigen.SpaceRecipesDeleteParams{SpaceSlug: args[0], RecipeId: args[1]}); err != nil {
				return runtime.NormalizeError(err)
			}
			if app.Config.JSON {
				return app.PrintJSON(map[string]any{"spaceSlug": args[0], "recipeId": args[1], "deleted": true})
			}
			app.Printf("Deleted recipe %s from %s\n", args[1], args[0])
			return nil
		}),
	}
}

func newActivityCmd(flags *support.RootFlags) *cobra.Command {
	page := pageFlags{}
	cmd := &cobra.Command{
		Use:   "activity <space> <recipe>",
		Short: "Show activity for a recipe",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			params := apigen.SpaceRecipeActivityListParams{SpaceSlug: args[0], RecipeId: args[1]}
			setPageParams(page.cursor, page.limit, &params.Cursor, &params.Limit)
			resp, err := api.SpaceRecipeActivityList(cmd.Context(), params)
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

func trimRequiredStrings(raw []string, field string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("at least one " + field + " is required")
	}
	values := make([]string, len(raw))
	for i, item := range raw {
		values[i] = strings.TrimSpace(item)
		if values[i] == "" {
			return nil, errors.New(field + " cannot be empty")
		}
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
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
