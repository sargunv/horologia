package recipecmd

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

const timeFormat = "2006-01-02 15:04:05Z07:00"

var (
	amountMixedPattern    = regexp.MustCompile(`^(\d+)\s+(\d+)/(\d+)(.*)$`)
	amountFractionPattern = regexp.MustCompile(`^(\d+)/(\d+)(.*)$`)
	amountDecimalPattern  = regexp.MustCompile(`^(\d+(?:\.\d+)?)(.*)$`)
	durationTokenPattern  = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(hours?|hrs?|h|minutes?|mins?|m)\s*`)
	itemLocatorPattern    = regexp.MustCompile(`^([1-9]\d*)\.([1-9]\d*)$`)
	yieldPattern          = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(.*)$`)
)

type pageFlags struct {
	cursor string
	limit  int32
}

type itemLocator struct {
	section int
	item    int
}

type parsedQuantity struct {
	quantity    *float64
	quantityMax *float64
	unit        string
}

func readRecipe(ctx context.Context, api *apigen.Client, spaceSlug string, recipeID string) (*apigen.Recipe, error) {
	return api.SpaceRecipesRead(ctx, apigen.SpaceRecipesReadParams{SpaceSlug: spaceSlug, RecipeId: recipeID})
}

func updateRecipe(ctx context.Context, api *apigen.Client, spaceSlug string, recipeID string, req *apigen.RecipeUpdate) (*apigen.Recipe, error) {
	return api.SpaceRecipesUpdate(ctx, req, apigen.SpaceRecipesUpdateParams{SpaceSlug: spaceSlug, RecipeId: recipeID})
}

func addPageFlags(cmd *cobra.Command, flags *pageFlags) {
	cmd.Flags().StringVar(&flags.cursor, "cursor", "", "Pagination cursor from a previous response")
	cmd.Flags().Int32Var(&flags.limit, "limit", 0, "Maximum number of items to return")
}

func setPageParams(cursor string, limit int32, cursorDst *apigen.OptString, limitDst *apigen.OptInt32) {
	if value := strings.TrimSpace(cursor); value != "" {
		cursorDst.SetTo(value)
	}
	if limit > 0 {
		limitDst.SetTo(limit)
	}
}

func parsePositiveIndex(raw string, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid %s %q; expected a positive integer", name, raw)
	}
	return value - 1, nil
}

func parseItemLocator(raw string) (itemLocator, error) {
	match := itemLocatorPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return itemLocator{}, fmt.Errorf("invalid locator %q; expected section.item, for example 1.2", raw)
	}
	section, sectionErr := strconv.Atoi(match[1])
	item, itemErr := strconv.Atoi(match[2])
	if sectionErr != nil || itemErr != nil {
		return itemLocator{}, fmt.Errorf("invalid locator %q; section and item are too large", raw)
	}
	return itemLocator{section: section - 1, item: item - 1}, nil
}

func parseAmountPrefix(input string) (float64, string, bool) {
	if match := amountMixedPattern.FindStringSubmatch(input); match != nil {
		whole, wholeOK := parseFiniteFloat(match[1])
		numerator, numeratorOK := parseFiniteFloat(match[2])
		denominator, denominatorOK := parseFiniteFloat(match[3])
		if !wholeOK || !numeratorOK || !denominatorOK || denominator == 0 {
			return 0, "", false
		}
		return whole + numerator/denominator, match[4], true
	}
	if match := amountFractionPattern.FindStringSubmatch(input); match != nil {
		numerator, numeratorOK := parseFiniteFloat(match[1])
		denominator, denominatorOK := parseFiniteFloat(match[2])
		if !numeratorOK || !denominatorOK || denominator == 0 {
			return 0, "", false
		}
		return numerator / denominator, match[3], true
	}
	if match := amountDecimalPattern.FindStringSubmatch(input); match != nil {
		value, ok := parseFiniteFloat(match[1])
		if !ok {
			return 0, "", false
		}
		return value, match[2], true
	}
	return 0, "", false
}

func parseFiniteFloat(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func parseQuantity(raw string) (parsedQuantity, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return parsedQuantity{}, fmt.Errorf("quantity cannot be empty")
	}

	first, rest, ok := parseAmountPrefix(value)
	if !ok {
		if value[0] >= '0' && value[0] <= '9' {
			return parsedQuantity{}, fmt.Errorf("invalid quantity %q", raw)
		}
		return parsedQuantity{unit: value}, nil
	}
	if first < 0.0001 {
		return parsedQuantity{}, fmt.Errorf("quantity must be at least 0.0001")
	}

	rest = strings.TrimSpace(rest)
	result := parsedQuantity{quantity: &first}
	if strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, "–") {
		rest = strings.TrimSpace(strings.TrimLeft(rest, "-–"))
		second, remaining, valid := parseAmountPrefix(rest)
		if !valid || second < first {
			return parsedQuantity{}, fmt.Errorf("invalid quantity range %q", raw)
		}
		result.quantityMax = &second
		rest = strings.TrimSpace(remaining)
	}
	if strings.HasPrefix(rest, "/") {
		return parsedQuantity{}, fmt.Errorf("invalid quantity %q", raw)
	}
	result.unit = rest
	return result, nil
}

func setIngredientQuantity(input *apigen.RecipeIngredientInput, quantity parsedQuantity) {
	input.Quantity.Reset()
	input.QuantityMax.Reset()
	input.Unit.Reset()
	if quantity.quantity != nil {
		input.Quantity.SetTo(*quantity.quantity)
	}
	if quantity.quantityMax != nil {
		input.QuantityMax.SetTo(*quantity.quantityMax)
	}
	if quantity.unit != "" {
		input.Unit.SetTo(quantity.unit)
	}
}

func parseDuration(raw string) (int32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}
	if minutes, err := strconv.ParseInt(value, 10, 32); err == nil {
		if minutes < 0 {
			return 0, fmt.Errorf("duration cannot be negative")
		}
		return int32(minutes), nil
	}

	remaining := value
	minutes := 0.0
	matched := false
	for remaining != "" {
		match := durationTokenPattern.FindStringSubmatch(remaining)
		if match == nil {
			return 0, fmt.Errorf("invalid duration %q; use minutes or values such as 1h 30m", raw)
		}
		amount, _ := strconv.ParseFloat(match[1], 64)
		unit := strings.ToLower(match[2])
		if strings.HasPrefix(unit, "h") {
			minutes += amount * 60
		} else {
			minutes += amount
		}
		remaining = strings.TrimSpace(strings.TrimPrefix(remaining, match[0]))
		matched = true
	}
	if !matched || minutes < 0 || minutes > math.MaxInt32 || minutes != math.Trunc(minutes) {
		return 0, fmt.Errorf("duration %q must resolve to a whole number of minutes", raw)
	}
	return int32(minutes), nil
}

func parseYield(raw string) (apigen.RecipeYield, error) {
	match := yieldPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return apigen.RecipeYield{}, fmt.Errorf("invalid yield %q; expected an amount and optional unit", raw)
	}
	amount, ok := parseFiniteFloat(match[1])
	if !ok || amount < 0.0001 {
		return apigen.RecipeYield{}, fmt.Errorf("yield amount must be at least 0.0001")
	}
	unit := strings.TrimSpace(match[2])
	if unit == "" {
		unit = "servings"
	}
	return apigen.RecipeYield{Amount: amount, Unit: unit}, nil
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatQuantity(ingredient apigen.RecipeIngredient) string {
	quantity, hasQuantity := ingredient.Quantity.Get()
	maximum, hasMaximum := ingredient.QuantityMax.Get()
	parts := make([]string, 0, 2)
	if hasQuantity {
		amount := formatNumber(quantity)
		if hasMaximum {
			amount += "–" + formatNumber(maximum)
		}
		parts = append(parts, amount)
	}
	if ingredient.Unit != "" {
		parts = append(parts, ingredient.Unit)
	}
	return strings.Join(parts, " ")
}

func formatDuration(minutes apigen.NilInt32) string {
	value, ok := minutes.Get()
	if !ok {
		return "none"
	}
	hours := value / 60
	remainder := value % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", remainder)
	}
	if remainder == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, remainder)
}

func formatYield(value apigen.NilRecipeYield) string {
	yield, ok := value.Get()
	if !ok {
		return "none"
	}
	return formatNumber(yield.Amount) + " " + yield.Unit
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func printRecipe(app *runtime.App, recipe *apigen.Recipe) {
	app.Printf("ID:          %s\n", recipe.ID)
	app.Printf("Space:       %s\n", recipe.SpaceSlug)
	app.Printf("Name:        %s\n", recipe.Name)
	app.Printf("Description: %s\n", recipe.Description)
	app.Printf("Yield:       %s\n", formatYield(recipe.Yield))
	app.Printf("Prep:        %s\n", formatDuration(recipe.PrepMinutes))
	app.Printf("Cook:        %s\n", formatDuration(recipe.CookMinutes))
	app.Printf("Tags:        %s\n", formatStringList(recipe.Tags))
	printIngredientSections(app, recipe.IngredientSections)
	printInstructionSections(app, recipe.InstructionSections)
	app.Printf("Created:     %s\n", recipe.CreatedAt.Format(timeFormat))
	app.Printf("Updated:     %s\n", recipe.UpdatedAt.Format(timeFormat))
}

func printRecipeList(app *runtime.App, recipes []apigen.RecipeSummary) error {
	if len(recipes) == 0 {
		app.Printf("No recipes.\n")
		return nil
	}
	w := tabwriter.NewWriter(app.Stdout, 0, 0, 2, ' ', 0)
	_, _ = w.Write([]byte("ID\tNAME\tSPACE\tYIELD\tPREP\tCOOK\n"))
	for _, recipe := range recipes {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", recipe.ID, recipe.Name, recipe.SpaceSlug, formatYield(recipe.Yield), formatDuration(recipe.PrepMinutes), formatDuration(recipe.CookMinutes))
	}
	return w.Flush()
}

func printIngredientSections(app *runtime.App, sections []apigen.RecipeIngredientSection) {
	app.Printf("Ingredients:\n")
	if len(sections) == 0 {
		app.Printf("  none\n")
		return
	}
	for sectionIndex, section := range sections {
		title := section.Title
		if title == "" {
			title = "(untitled)"
		}
		app.Printf("  [%d] %s\n", sectionIndex+1, title)
		for itemIndex, ingredient := range section.Ingredients {
			quantity := formatQuantity(ingredient)
			if quantity == "" {
				app.Printf("    %d.%d  %s\n", sectionIndex+1, itemIndex+1, ingredient.Item)
			} else {
				app.Printf("    %d.%d  %-12s %s\n", sectionIndex+1, itemIndex+1, quantity, ingredient.Item)
			}
		}
	}
}

func printInstructionSections(app *runtime.App, sections []apigen.RecipeInstructionSection) {
	app.Printf("Instructions:\n")
	if len(sections) == 0 {
		app.Printf("  none\n")
		return
	}
	for sectionIndex, section := range sections {
		title := section.Title
		if title == "" {
			title = "(untitled)"
		}
		app.Printf("  [%d] %s\n", sectionIndex+1, title)
		for itemIndex, step := range section.Steps {
			app.Printf("    %d.%d  %s\n", sectionIndex+1, itemIndex+1, step.Body)
		}
	}
}

func ingredientSectionsInput(sections []apigen.RecipeIngredientSection) []apigen.RecipeIngredientSectionInput {
	result := make([]apigen.RecipeIngredientSectionInput, len(sections))
	for sectionIndex, section := range sections {
		if section.Title != "" {
			result[sectionIndex].Title.SetTo(section.Title)
		}
		result[sectionIndex].Ingredients = make([]apigen.RecipeIngredientInput, len(section.Ingredients))
		for itemIndex, ingredient := range section.Ingredients {
			input := apigen.RecipeIngredientInput{Item: ingredient.Item}
			if value, ok := ingredient.Quantity.Get(); ok {
				input.Quantity.SetTo(value)
			}
			if value, ok := ingredient.QuantityMax.Get(); ok {
				input.QuantityMax.SetTo(value)
			}
			if ingredient.Unit != "" {
				input.Unit.SetTo(ingredient.Unit)
			}
			result[sectionIndex].Ingredients[itemIndex] = input
		}
	}
	return result
}

func instructionSectionsInput(sections []apigen.RecipeInstructionSection) []apigen.RecipeInstructionSectionInput {
	result := make([]apigen.RecipeInstructionSectionInput, len(sections))
	for sectionIndex, section := range sections {
		if section.Title != "" {
			result[sectionIndex].Title.SetTo(section.Title)
		}
		result[sectionIndex].Steps = make([]apigen.RecipeStepInput, len(section.Steps))
		for itemIndex, step := range section.Steps {
			result[sectionIndex].Steps[itemIndex] = apigen.RecipeStepInput(step)
		}
	}
	return result
}

func insertAt[T any](items []T, index int, value T) ([]T, error) {
	if index < 0 || index > len(items) {
		return nil, fmt.Errorf("position must be between 1 and %d", len(items)+1)
	}
	items = append(items, value)
	copy(items[index+1:], items[index:len(items)-1])
	items[index] = value
	return items, nil
}

func moveAt[T any](items []T, from int, to int) ([]T, error) {
	if from < 0 || from >= len(items) {
		return nil, fmt.Errorf("source position is out of range")
	}
	value := items[from]
	items = append(items[:from], items[from+1:]...)
	return insertAt(items, to, value)
}

func printActivityPage(app *runtime.App, page *apigen.ActivityLogPage) {
	if len(page.Items) == 0 {
		app.Printf("No activity.\n")
	} else {
		for _, entry := range page.Items {
			app.Printf("%s %s %s %s\n", entry.CreatedAt.Format(timeFormat), entry.EntityType, entry.EntityId, entry.Action)
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
