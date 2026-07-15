package recipecmd

import (
	"errors"
	"fmt"
	"strings"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/cmd/support"
	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func newIngredientCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("ingredient", "Manage recipe ingredients")
	cmd.AddCommand(
		newIngredientListCmd(flags),
		newIngredientAddCmd(flags),
		newIngredientUpdateCmd(flags),
		newIngredientRemoveCmd(flags),
		newIngredientMoveCmd(flags),
		newIngredientClearCmd(flags),
		newIngredientSectionCmd(flags),
	)
	return cmd
}

func newIngredientListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space> <recipe>",
		Short: "List recipe ingredients",
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
				return app.PrintJSON(recipe.IngredientSections)
			}
			printIngredientSections(app, recipe.IngredientSections)
			return nil
		}),
	}
}

func newIngredientAddCmd(flags *support.RootFlags) *cobra.Command {
	var ingredient string
	var quantity string
	var section int
	var position int
	cmd := &cobra.Command{
		Use:   "add <space> <recipe>",
		Short: "Add a recipe ingredient",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			item := strings.TrimSpace(ingredient)
			if item == "" {
				return errors.New("ingredient cannot be empty")
			}
			input := apigen.RecipeIngredientInput{Item: item}
			if cmd.Flags().Changed("quantity") {
				parsed, err := parseQuantity(quantity)
				if err != nil {
					return err
				}
				setIngredientQuantity(&input, parsed)
			}

			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			if len(sections) == 0 {
				if cmd.Flags().Changed("section") && section != 1 {
					return errors.New("section must be 1 when adding the first ingredient")
				}
				sections = []apigen.RecipeIngredientSectionInput{{Ingredients: []apigen.RecipeIngredientInput{}}}
			}
			sectionIndex, err := selectSection(len(sections), section, cmd.Flags().Changed("section"))
			if err != nil {
				return err
			}
			positionIndex := len(sections[sectionIndex].Ingredients)
			if cmd.Flags().Changed("position") {
				positionIndex = position - 1
			}
			sections[sectionIndex].Ingredients, err = insertAt(sections[sectionIndex].Ingredients, positionIndex, input)
			if err != nil {
				return err
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&ingredient, "ingredient", "", "Ingredient name")
	cmd.Flags().StringVarP(&quantity, "quantity", "q", "", "Quantity and unit, for example '1 1/2 cups'")
	cmd.Flags().IntVar(&section, "section", 0, "One-based destination section")
	cmd.Flags().IntVar(&position, "position", 0, "One-based position within the section")
	_ = cmd.MarkFlagRequired("ingredient")
	return cmd
}

func newIngredientUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var ingredient string
	var quantity string
	var clearQuantity bool
	cmd := &cobra.Command{
		Use:   "update <space> <recipe> <locator>",
		Short: "Update a recipe ingredient",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			locator, err := parseItemLocator(args[2])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("quantity") && clearQuantity {
				return errors.New("quantity and clear-quantity cannot be used together")
			}
			if !cmd.Flags().Changed("ingredient") && !cmd.Flags().Changed("quantity") && !clearQuantity {
				return errors.New("at least one field flag is required")
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			input, err := ingredientAt(sections, locator)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("ingredient") {
				item := strings.TrimSpace(ingredient)
				if item == "" {
					return errors.New("ingredient cannot be empty")
				}
				input.Item = item
			}
			if cmd.Flags().Changed("quantity") {
				parsed, err := parseQuantity(quantity)
				if err != nil {
					return err
				}
				setIngredientQuantity(input, parsed)
			} else if clearQuantity {
				setIngredientQuantity(input, parsedQuantity{})
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&ingredient, "ingredient", "", "Updated ingredient name")
	cmd.Flags().StringVarP(&quantity, "quantity", "q", "", "Updated quantity and unit")
	cmd.Flags().BoolVar(&clearQuantity, "clear-quantity", false, "Clear the ingredient quantity and unit")
	return cmd
}

func newIngredientRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <recipe> <locator>",
		Short: "Remove a recipe ingredient",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			locator, err := parseItemLocator(args[2])
			if err != nil {
				return err
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			if _, err := ingredientAt(sections, locator); err != nil {
				return err
			}
			items := sections[locator.section].Ingredients
			sections[locator.section].Ingredients = append(items[:locator.item], items[locator.item+1:]...)
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
}

func newIngredientMoveCmd(flags *support.RootFlags) *cobra.Command {
	var section int
	var position int
	cmd := &cobra.Command{
		Use:   "move <space> <recipe> <locator>",
		Short: "Move a recipe ingredient",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			locator, err := parseItemLocator(args[2])
			if err != nil {
				return err
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			input, err := ingredientAt(sections, locator)
			if err != nil {
				return err
			}
			value := *input
			targetSection := locator.section
			if cmd.Flags().Changed("section") {
				targetSection, err = selectSection(len(sections), section, true)
				if err != nil {
					return err
				}
			}
			sourceItems := sections[locator.section].Ingredients
			sections[locator.section].Ingredients = append(sourceItems[:locator.item], sourceItems[locator.item+1:]...)
			sections[targetSection].Ingredients, err = insertAt(sections[targetSection].Ingredients, position-1, value)
			if err != nil {
				return err
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().IntVar(&section, "section", 0, "One-based destination section (defaults to the current section)")
	cmd.Flags().IntVar(&position, "position", 0, "Final one-based position within the destination section")
	_ = cmd.MarkFlagRequired("position")
	return cmd
}

func newIngredientClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <recipe>",
		Short: "Clear all recipe ingredient sections",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], []apigen.RecipeIngredientSectionInput{})
		}),
	}
}

func newIngredientSectionCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("section", "Manage recipe ingredient sections")
	cmd.AddCommand(
		newIngredientSectionAddCmd(flags),
		newIngredientSectionUpdateCmd(flags),
		newIngredientSectionRemoveCmd(flags),
		newIngredientSectionMoveCmd(flags),
	)
	return cmd
}

func newIngredientSectionAddCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var position int
	cmd := &cobra.Command{
		Use:   "add <space> <recipe>",
		Short: "Add an ingredient section",
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
			sections := ingredientSectionsInput(recipe.IngredientSections)
			section := apigen.RecipeIngredientSectionInput{Ingredients: []apigen.RecipeIngredientInput{}}
			if value := strings.TrimSpace(title); value != "" {
				section.Title.SetTo(value)
			}
			index := len(sections)
			if cmd.Flags().Changed("position") {
				index = position - 1
			}
			sections, err = insertAt(sections, index, section)
			if err != nil {
				return err
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&title, "title", "", "Section title")
	cmd.Flags().IntVar(&position, "position", 0, "One-based section position")
	return cmd
}

func newIngredientSectionUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var clearTitle bool
	cmd := &cobra.Command{
		Use:   "update <space> <recipe> <section>",
		Short: "Update an ingredient section",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("title") == clearTitle {
				return errors.New("specify exactly one of title or clear-title")
			}
			sectionIndex, err := parsePositiveIndex(args[2], "section")
			if err != nil {
				return err
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			if err := validateSectionIndex(len(sections), sectionIndex); err != nil {
				return err
			}
			sections[sectionIndex].Title.Reset()
			if !clearTitle {
				value := strings.TrimSpace(title)
				if value == "" {
					return errors.New("section title cannot be empty; use clear-title")
				}
				sections[sectionIndex].Title.SetTo(value)
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&title, "title", "", "Updated section title")
	cmd.Flags().BoolVar(&clearTitle, "clear-title", false, "Clear the section title")
	return cmd
}

func newIngredientSectionRemoveCmd(flags *support.RootFlags) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <space> <recipe> <section>",
		Short: "Remove an ingredient section",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			sectionIndex, err := parsePositiveIndex(args[2], "section")
			if err != nil {
				return err
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			if err := validateSectionIndex(len(sections), sectionIndex); err != nil {
				return err
			}
			if len(sections[sectionIndex].Ingredients) > 0 && !force {
				return errors.New("section is not empty; use force to remove it and its ingredients")
			}
			sections = append(sections[:sectionIndex], sections[sectionIndex+1:]...)
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove a non-empty section and its ingredients")
	return cmd
}

func newIngredientSectionMoveCmd(flags *support.RootFlags) *cobra.Command {
	var position int
	cmd := &cobra.Command{
		Use:   "move <space> <recipe> <section>",
		Short: "Move an ingredient section",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			sectionIndex, err := parsePositiveIndex(args[2], "section")
			if err != nil {
				return err
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := ingredientSectionsInput(recipe.IngredientSections)
			sections, err = moveAt(sections, sectionIndex, position-1)
			if err != nil {
				return err
			}
			return patchIngredientSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().IntVar(&position, "position", 0, "Final one-based section position")
	_ = cmd.MarkFlagRequired("position")
	return cmd
}

func ingredientAt(sections []apigen.RecipeIngredientSectionInput, locator itemLocator) (*apigen.RecipeIngredientInput, error) {
	if err := validateSectionIndex(len(sections), locator.section); err != nil {
		return nil, err
	}
	if locator.item < 0 || locator.item >= len(sections[locator.section].Ingredients) {
		return nil, fmt.Errorf("ingredient %d.%d does not exist", locator.section+1, locator.item+1)
	}
	return &sections[locator.section].Ingredients[locator.item], nil
}

func selectSection(count int, raw int, explicit bool) (int, error) {
	if explicit {
		index := raw - 1
		if err := validateSectionIndex(count, index); err != nil {
			return 0, err
		}
		return index, nil
	}
	if count == 1 {
		return 0, nil
	}
	return 0, fmt.Errorf("recipe has %d sections; select one with --section", count)
}

func validateSectionIndex(count int, index int) error {
	if index < 0 || index >= count {
		return fmt.Errorf("section %d does not exist", index+1)
	}
	return nil
}

func patchIngredientSections(app *runtime.App, cmd *cobra.Command, api *apigen.Client, spaceSlug string, recipeID string, sections []apigen.RecipeIngredientSectionInput) error {
	recipe, err := updateRecipe(cmd.Context(), api, spaceSlug, recipeID, &apigen.RecipeUpdate{IngredientSections: sections})
	if err != nil {
		return runtime.NormalizeError(err)
	}
	return printMutationResult(app, recipe)
}
