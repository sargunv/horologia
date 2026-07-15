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

func newInstructionCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("instruction", "Manage recipe instructions")
	cmd.AddCommand(
		newInstructionListCmd(flags),
		newInstructionAddCmd(flags),
		newInstructionUpdateCmd(flags),
		newInstructionRemoveCmd(flags),
		newInstructionMoveCmd(flags),
		newInstructionClearCmd(flags),
		newInstructionSectionCmd(flags),
	)
	return cmd
}

func newInstructionListCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <space> <recipe>",
		Short: "List recipe instructions",
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
				return app.PrintJSON(recipe.InstructionSections)
			}
			printInstructionSections(app, recipe.InstructionSections)
			return nil
		}),
	}
}

func newInstructionAddCmd(flags *support.RootFlags) *cobra.Command {
	var body string
	var section int
	var position int
	cmd := &cobra.Command{
		Use:   "add <space> <recipe>",
		Short: "Add a recipe instruction",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(body) == "" {
				return errors.New("instruction body cannot be empty")
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := instructionSectionsInput(recipe.InstructionSections)
			if len(sections) == 0 {
				if cmd.Flags().Changed("section") && section != 1 {
					return errors.New("section must be 1 when adding the first instruction")
				}
				sections = []apigen.RecipeInstructionSectionInput{{Steps: []apigen.RecipeStepInput{}}}
			}
			sectionIndex, err := selectSection(len(sections), section, cmd.Flags().Changed("section"))
			if err != nil {
				return err
			}
			positionIndex := len(sections[sectionIndex].Steps)
			if cmd.Flags().Changed("position") {
				positionIndex = position - 1
			}
			sections[sectionIndex].Steps, err = insertAt(sections[sectionIndex].Steps, positionIndex, apigen.RecipeStepInput{Body: body})
			if err != nil {
				return err
			}
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&body, "body", "", "Instruction body")
	cmd.Flags().IntVar(&section, "section", 0, "One-based destination section")
	cmd.Flags().IntVar(&position, "position", 0, "One-based position within the section")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newInstructionUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var body string
	cmd := &cobra.Command{
		Use:   "update <space> <recipe> <locator>",
		Short: "Update a recipe instruction",
		Args:  cobra.ExactArgs(3),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			locator, err := parseItemLocator(args[2])
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return errors.New("instruction body cannot be empty")
			}
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			recipe, err := readRecipe(cmd.Context(), api, args[0], args[1])
			if err != nil {
				return runtime.NormalizeError(err)
			}
			sections := instructionSectionsInput(recipe.InstructionSections)
			step, err := instructionAt(sections, locator)
			if err != nil {
				return err
			}
			step.Body = body
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&body, "body", "", "Updated instruction body")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newInstructionRemoveCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <space> <recipe> <locator>",
		Short: "Remove a recipe instruction",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
			if _, err := instructionAt(sections, locator); err != nil {
				return err
			}
			steps := sections[locator.section].Steps
			sections[locator.section].Steps = append(steps[:locator.item], steps[locator.item+1:]...)
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
}

func newInstructionMoveCmd(flags *support.RootFlags) *cobra.Command {
	var section int
	var position int
	cmd := &cobra.Command{
		Use:   "move <space> <recipe> <locator>",
		Short: "Move a recipe instruction",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
			step, err := instructionAt(sections, locator)
			if err != nil {
				return err
			}
			value := *step
			targetSection := locator.section
			if cmd.Flags().Changed("section") {
				targetSection, err = selectSection(len(sections), section, true)
				if err != nil {
					return err
				}
			}
			sourceSteps := sections[locator.section].Steps
			sections[locator.section].Steps = append(sourceSteps[:locator.item], sourceSteps[locator.item+1:]...)
			sections[targetSection].Steps, err = insertAt(sections[targetSection].Steps, position-1, value)
			if err != nil {
				return err
			}
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().IntVar(&section, "section", 0, "One-based destination section (defaults to the current section)")
	cmd.Flags().IntVar(&position, "position", 0, "Final one-based position within the destination section")
	_ = cmd.MarkFlagRequired("position")
	return cmd
}

func newInstructionClearCmd(flags *support.RootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <space> <recipe>",
		Short: "Clear all recipe instruction sections",
		Args:  cobra.ExactArgs(2),
		RunE: support.RunWithApp(flags, func(app *runtime.App, cmd *cobra.Command, args []string) error {
			api, err := support.RequireAPI(app)
			if err != nil {
				return err
			}
			return patchInstructionSections(app, cmd, api, args[0], args[1], []apigen.RecipeInstructionSectionInput{})
		}),
	}
}

func newInstructionSectionCmd(flags *support.RootFlags) *cobra.Command {
	cmd := support.GroupCommand("section", "Manage recipe instruction sections")
	cmd.AddCommand(
		newInstructionSectionAddCmd(flags),
		newInstructionSectionUpdateCmd(flags),
		newInstructionSectionRemoveCmd(flags),
		newInstructionSectionMoveCmd(flags),
	)
	return cmd
}

func newInstructionSectionAddCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var position int
	cmd := &cobra.Command{
		Use:   "add <space> <recipe>",
		Short: "Add an instruction section",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
			section := apigen.RecipeInstructionSectionInput{Steps: []apigen.RecipeStepInput{}}
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
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&title, "title", "", "Section title")
	cmd.Flags().IntVar(&position, "position", 0, "One-based section position")
	return cmd
}

func newInstructionSectionUpdateCmd(flags *support.RootFlags) *cobra.Command {
	var title string
	var clearTitle bool
	cmd := &cobra.Command{
		Use:   "update <space> <recipe> <section>",
		Short: "Update an instruction section",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
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
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().StringVar(&title, "title", "", "Updated section title")
	cmd.Flags().BoolVar(&clearTitle, "clear-title", false, "Clear the section title")
	return cmd
}

func newInstructionSectionRemoveCmd(flags *support.RootFlags) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <space> <recipe> <section>",
		Short: "Remove an instruction section",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
			if err := validateSectionIndex(len(sections), sectionIndex); err != nil {
				return err
			}
			if len(sections[sectionIndex].Steps) > 0 && !force {
				return errors.New("section is not empty; use force to remove it and its instructions")
			}
			sections = append(sections[:sectionIndex], sections[sectionIndex+1:]...)
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove a non-empty section and its instructions")
	return cmd
}

func newInstructionSectionMoveCmd(flags *support.RootFlags) *cobra.Command {
	var position int
	cmd := &cobra.Command{
		Use:   "move <space> <recipe> <section>",
		Short: "Move an instruction section",
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
			sections := instructionSectionsInput(recipe.InstructionSections)
			sections, err = moveAt(sections, sectionIndex, position-1)
			if err != nil {
				return err
			}
			return patchInstructionSections(app, cmd, api, args[0], args[1], sections)
		}),
	}
	cmd.Flags().IntVar(&position, "position", 0, "Final one-based section position")
	_ = cmd.MarkFlagRequired("position")
	return cmd
}

func instructionAt(sections []apigen.RecipeInstructionSectionInput, locator itemLocator) (*apigen.RecipeStepInput, error) {
	if err := validateSectionIndex(len(sections), locator.section); err != nil {
		return nil, err
	}
	if locator.item < 0 || locator.item >= len(sections[locator.section].Steps) {
		return nil, fmt.Errorf("instruction %d.%d does not exist", locator.section+1, locator.item+1)
	}
	return &sections[locator.section].Steps[locator.item], nil
}

func patchInstructionSections(app *runtime.App, cmd *cobra.Command, api *apigen.Client, spaceSlug string, recipeID string, sections []apigen.RecipeInstructionSectionInput) error {
	recipe, err := updateRecipe(cmd.Context(), api, spaceSlug, recipeID, &apigen.RecipeUpdate{InstructionSections: sections})
	if err != nil {
		return runtime.NormalizeError(err)
	}
	return printMutationResult(app, recipe)
}
