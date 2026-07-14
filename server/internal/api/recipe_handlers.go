package api

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/sargunv/horologia/server/internal/activitylog"
	"github.com/sargunv/horologia/server/internal/auth"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/tagname"
	"github.com/sargunv/horologia/server/internal/types"
)

var exactRecipeIDPattern = regexp.MustCompile(`^(?:R[1-9]\d*|[1-9]\d*)$`)

type recipeListCursor struct {
	UpdatedAt pgtype.Timestamptz
	ID        int64
}

func encodeRecipeListCursor(row dbgen.Recipe) string {
	return row.UpdatedAt.Time.Format(time.RFC3339Nano) + "~" + strconv.FormatInt(row.ID, 10)
}

func decodeRecipeListCursor(opt apigen.OptString) (recipeListCursor, error) {
	if !opt.IsSet() {
		return recipeListCursor{}, nil
	}
	raw, err := decodeCursor(opt)
	if err != nil {
		return recipeListCursor{}, err
	}
	parts := strings.Split(raw, "~")
	if len(parts) != 2 {
		return recipeListCursor{}, fmt.Errorf("invalid cursor: expected 2 parts, got %d", len(parts))
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return recipeListCursor{}, fmt.Errorf("invalid cursor updated_at: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return recipeListCursor{}, fmt.Errorf("invalid cursor id: %w", err)
	}
	return recipeListCursor{
		UpdatedAt: types.Timestamptz(updatedAt),
		ID:        id,
	}, nil
}

func numericFromFloat64(value float64) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	if err := result.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("convert number: %w", err)
	}
	return result, nil
}

func nilFloat64FromNumeric(value pgtype.Numeric) (apigen.NilFloat64, error) {
	if !value.Valid {
		return apigen.NilFloat64{Null: true}, nil
	}
	converted, err := value.Float64Value()
	if err != nil {
		return apigen.NilFloat64{}, fmt.Errorf("convert numeric: %w", err)
	}
	if !converted.Valid {
		return apigen.NilFloat64{Null: true}, nil
	}
	return apigen.NewNilFloat64(converted.Float64), nil
}

func nilInt32FromDB(value pgtype.Int4) apigen.NilInt32 {
	if !value.Valid {
		return apigen.NilInt32{Null: true}
	}
	return apigen.NewNilInt32(value.Int32)
}

func recipeYieldFromDB(amount pgtype.Numeric, unit pgtype.Text) (apigen.NilRecipeYield, error) {
	if !amount.Valid {
		return apigen.NilRecipeYield{Null: true}, nil
	}
	converted, err := amount.Float64Value()
	if err != nil {
		return apigen.NilRecipeYield{}, fmt.Errorf("convert recipe yield: %w", err)
	}
	return apigen.NewNilRecipeYield(apigen.RecipeYield{
		Amount: converted.Float64,
		Unit:   unit.String,
	}), nil
}

func recipeYieldToDB(value apigen.RecipeYield) (pgtype.Numeric, pgtype.Text, error) {
	if value.Amount <= 0 {
		return pgtype.Numeric{}, pgtype.Text{}, badRequest("yield amount must be greater than zero")
	}
	if strings.TrimSpace(value.Unit) == "" {
		return pgtype.Numeric{}, pgtype.Text{}, badRequest("yield unit cannot be empty")
	}
	amount, err := numericFromFloat64(value.Amount)
	if err != nil {
		return pgtype.Numeric{}, pgtype.Text{}, err
	}
	return amount, pgtype.Text{String: value.Unit, Valid: true}, nil
}

func recipeSummaryFromDB(recipe dbgen.Recipe, tagNames []string) (*apigen.RecipeSummary, error) {
	yield, err := recipeYieldFromDB(recipe.YieldAmount, recipe.YieldUnit)
	if err != nil {
		return nil, err
	}
	return &apigen.RecipeSummary{
		ID:          types.FormatRecipeID(recipe.ID),
		SpaceSlug:   recipe.SpaceSlug,
		Name:        recipe.Name,
		Yield:       yield,
		PrepMinutes: nilInt32FromDB(recipe.PrepMinutes),
		CookMinutes: nilInt32FromDB(recipe.CookMinutes),
		Tags:        tagNames,
		UpdatedAt:   tsToTime(recipe.UpdatedAt),
	}, nil
}

func recipeFromSearchRow(row dbgen.SearchVisibleRecipesRow) dbgen.Recipe {
	return dbgen.Recipe(row)
}

func fetchRecipeIngredientSections(ctx context.Context, q *dbgen.Queries, recipeID int64) ([]apigen.RecipeIngredientSection, error) {
	rows, err := q.ListRecipeIngredientRows(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	sections := make([]apigen.RecipeIngredientSection, 0)
	var currentSectionID int64
	for _, row := range rows {
		if len(sections) == 0 || row.SectionID != currentSectionID {
			currentSectionID = row.SectionID
			sections = append(sections, apigen.RecipeIngredientSection{
				Title:       row.SectionTitle,
				Ingredients: []apigen.RecipeIngredient{},
			})
		}
		if !row.IngredientID.Valid {
			continue
		}
		quantity, err := nilFloat64FromNumeric(row.Quantity)
		if err != nil {
			return nil, err
		}
		quantityMax, err := nilFloat64FromNumeric(row.QuantityMax)
		if err != nil {
			return nil, err
		}
		section := &sections[len(sections)-1]
		section.Ingredients = append(section.Ingredients, apigen.RecipeIngredient{
			Quantity:    quantity,
			QuantityMax: quantityMax,
			Unit:        row.Unit.String,
			Item:        row.Item.String,
			Preparation: row.Preparation.String,
			Optional:    row.Optional.Bool,
		})
	}
	return sections, nil
}

func fetchRecipeInstructionSections(ctx context.Context, q *dbgen.Queries, recipeID int64) ([]apigen.RecipeInstructionSection, error) {
	rows, err := q.ListRecipeInstructionRows(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	sections := make([]apigen.RecipeInstructionSection, 0)
	var currentSectionID int64
	for _, row := range rows {
		if len(sections) == 0 || row.SectionID != currentSectionID {
			currentSectionID = row.SectionID
			sections = append(sections, apigen.RecipeInstructionSection{
				Title: row.SectionTitle,
				Steps: []apigen.RecipeStep{},
			})
		}
		if !row.StepID.Valid {
			continue
		}
		section := &sections[len(sections)-1]
		section.Steps = append(section.Steps, apigen.RecipeStep{Body: row.Body.String})
	}
	return sections, nil
}

func (h *Handler) fetchRecipe(ctx context.Context, q *dbgen.Queries, id int64, spaceSlug string) (*apigen.Recipe, error) {
	recipe, err := q.GetRecipe(ctx, dbgen.GetRecipeParams{ID: id, SpaceSlug: spaceSlug})
	if err != nil {
		return nil, err
	}
	tagNames, err := q.ListTagNamesByRecipe(ctx, id)
	if err != nil {
		return nil, err
	}
	ingredientSections, err := fetchRecipeIngredientSections(ctx, q, id)
	if err != nil {
		return nil, err
	}
	instructionSections, err := fetchRecipeInstructionSections(ctx, q, id)
	if err != nil {
		return nil, err
	}
	yield, err := recipeYieldFromDB(recipe.YieldAmount, recipe.YieldUnit)
	if err != nil {
		return nil, err
	}
	return &apigen.Recipe{
		ID:                  types.FormatRecipeID(recipe.ID),
		SpaceSlug:           recipe.SpaceSlug,
		Name:                recipe.Name,
		Description:         recipe.Description,
		Yield:               yield,
		PrepMinutes:         nilInt32FromDB(recipe.PrepMinutes),
		CookMinutes:         nilInt32FromDB(recipe.CookMinutes),
		Source:              recipe.Source,
		SourceUrl:           nilStringFromDB(recipe.SourceUrl),
		Tags:                tagNames,
		IngredientSections:  ingredientSections,
		InstructionSections: instructionSections,
		CreatedAt:           tsToTime(recipe.CreatedAt),
		UpdatedAt:           tsToTime(recipe.UpdatedAt),
	}, nil
}

func (h *Handler) enrichRecipeSummaries(ctx context.Context, q *dbgen.Queries, recipes []dbgen.Recipe) ([]apigen.RecipeSummary, error) {
	if len(recipes) == 0 {
		return []apigen.RecipeSummary{}, nil
	}
	ids := make([]int64, len(recipes))
	for i, recipe := range recipes {
		ids[i] = recipe.ID
	}
	tagRows, err := q.ListTagNamesByRecipes(ctx, ids)
	if err != nil {
		return nil, err
	}
	tagMap := make(map[int64][]string)
	for _, row := range tagRows {
		tagMap[row.RecipeID] = append(tagMap[row.RecipeID], row.Name)
	}
	result := make([]apigen.RecipeSummary, 0, len(recipes))
	for _, recipe := range recipes {
		tagNames := tagMap[recipe.ID]
		if tagNames == nil {
			tagNames = []string{}
		}
		summary, err := recipeSummaryFromDB(recipe, tagNames)
		if err != nil {
			return nil, err
		}
		result = append(result, *summary)
	}
	return result, nil
}

func validateRecipeName(name string) error {
	if strings.TrimSpace(name) == "" {
		return badRequest("recipe name cannot be empty")
	}
	return nil
}

func validateRecipeCollections(ingredientSections []apigen.RecipeIngredientSectionInput, instructionSections []apigen.RecipeInstructionSectionInput) error {
	if len(ingredientSections) > 20 || len(instructionSections) > 20 {
		return badRequest("recipes may have at most 20 ingredient and instruction sections")
	}
	for _, section := range ingredientSections {
		if len(section.Ingredients) > 100 {
			return badRequest("ingredient sections may have at most 100 ingredients")
		}
		for _, ingredient := range section.Ingredients {
			if strings.TrimSpace(ingredient.Item) == "" {
				return badRequest("ingredient item cannot be empty")
			}
			quantity := ingredient.Quantity.Or(0)
			quantityMax := ingredient.QuantityMax.Or(0)
			if ingredient.Quantity.IsSet() && quantity <= 0 {
				return badRequest("ingredient quantity must be greater than zero")
			}
			if ingredient.QuantityMax.IsSet() {
				if !ingredient.Quantity.IsSet() {
					return badRequest("ingredient quantityMax requires quantity")
				}
				if quantityMax < quantity {
					return badRequest("ingredient quantityMax must be at least quantity")
				}
			}
		}
	}
	for _, section := range instructionSections {
		if len(section.Steps) > 100 {
			return badRequest("instruction sections may have at most 100 steps")
		}
		for _, step := range section.Steps {
			if strings.TrimSpace(step.Body) == "" {
				return badRequest("recipe step cannot be empty")
			}
		}
	}
	return nil
}

func setRecipeIngredientSections(ctx context.Context, q *dbgen.Queries, recipeID int64, sections []apigen.RecipeIngredientSectionInput) error {
	if err := q.DeleteRecipeIngredientSections(ctx, recipeID); err != nil {
		return err
	}
	for sectionPosition, inputSection := range sections {
		section, err := q.InsertRecipeIngredientSection(ctx, dbgen.InsertRecipeIngredientSectionParams{
			RecipeID: recipeID,
			Title:    inputSection.Title.Or(""),
			Position: int32(sectionPosition),
		})
		if err != nil {
			return err
		}
		for ingredientPosition, ingredient := range inputSection.Ingredients {
			var quantity, quantityMax pgtype.Numeric
			if ingredient.Quantity.IsSet() {
				quantity, err = numericFromFloat64(ingredient.Quantity.Value)
				if err != nil {
					return err
				}
			}
			if ingredient.QuantityMax.IsSet() {
				quantityMax, err = numericFromFloat64(ingredient.QuantityMax.Value)
				if err != nil {
					return err
				}
			}
			if err := q.InsertRecipeIngredient(ctx, dbgen.InsertRecipeIngredientParams{
				SectionID:   section.ID,
				Position:    int32(ingredientPosition),
				Quantity:    quantity,
				QuantityMax: quantityMax,
				Unit:        ingredient.Unit.Or(""),
				Item:        ingredient.Item,
				Preparation: ingredient.Preparation.Or(""),
				Optional:    ingredient.Optional.Or(false),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func setRecipeInstructionSections(ctx context.Context, q *dbgen.Queries, recipeID int64, sections []apigen.RecipeInstructionSectionInput) error {
	if err := q.DeleteRecipeInstructionSections(ctx, recipeID); err != nil {
		return err
	}
	for sectionPosition, inputSection := range sections {
		section, err := q.InsertRecipeInstructionSection(ctx, dbgen.InsertRecipeInstructionSectionParams{
			RecipeID: recipeID,
			Title:    inputSection.Title.Or(""),
			Position: int32(sectionPosition),
		})
		if err != nil {
			return err
		}
		for stepPosition, step := range inputSection.Steps {
			if err := q.InsertRecipeStep(ctx, dbgen.InsertRecipeStepParams{
				SectionID: section.ID,
				Position:  int32(stepPosition),
				Body:      step.Body,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func setRecipeTags(ctx context.Context, q *dbgen.Queries, recipeID int64, spaceSlug string, tagNames []string, now time.Time) error {
	if err := q.DeleteRecipeTags(ctx, recipeID); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(tagNames))
	for _, name := range tagNames {
		if err := validateTagName(name); err != nil {
			return err
		}
		folded := tagname.Fold(name)
		if _, ok := seen[folded]; ok {
			continue
		}
		seen[folded] = struct{}{}
		tag, err := q.EnsureTag(ctx, dbgen.EnsureTagParams{
			SpaceSlug:  spaceSlug,
			Name:       name,
			NameFolded: folded,
			CreatedAt:  types.Timestamptz(now),
		})
		if err != nil {
			return err
		}
		if err := q.InsertRecipeTag(ctx, dbgen.InsertRecipeTagParams{
			RecipeID:  recipeID,
			TagID:     tag.ID,
			SpaceSlug: spaceSlug,
			CreatedAt: types.Timestamptz(now),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) SpaceRecipesCreate(ctx context.Context, req *apigen.RecipeCreate, params apigen.SpaceRecipesCreateParams) (*apigen.Recipe, error) {
	if err := h.requireScope(ctx, "recipes:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	if err := validateRecipeName(req.Name); err != nil {
		return nil, err
	}
	if err := validateRecipeCollections(req.IngredientSections, req.InstructionSections); err != nil {
		return nil, err
	}

	var yieldAmount pgtype.Numeric
	var yieldUnit pgtype.Text
	var err error
	if req.Yield.IsSet() {
		yieldAmount, yieldUnit, err = recipeYieldToDB(req.Yield.Value)
		if err != nil {
			return nil, err
		}
	}
	var prepMinutes, cookMinutes pgtype.Int4
	if req.PrepMinutes.IsSet() {
		prepMinutes = pgtype.Int4{Int32: req.PrepMinutes.Value, Valid: true}
	}
	if req.CookMinutes.IsSet() {
		cookMinutes = pgtype.Int4{Int32: req.CookMinutes.Value, Valid: true}
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)
	now := time.Now()
	recipe, err := q.CreateRecipe(ctx, dbgen.CreateRecipeParams{
		SpaceSlug:   params.SpaceSlug,
		Name:        req.Name,
		Description: req.Description.Or(""),
		YieldAmount: yieldAmount,
		YieldUnit:   yieldUnit,
		PrepMinutes: prepMinutes,
		CookMinutes: cookMinutes,
		Source:      req.Source.Or(""),
		SourceUrl:   optStringToDB(req.SourceUrl),
		CreatedAt:   types.Timestamptz(now),
		UpdatedAt:   types.Timestamptz(now),
	})
	if err != nil {
		return nil, err
	}
	if err := setRecipeIngredientSections(ctx, q, recipe.ID, req.IngredientSections); err != nil {
		return nil, err
	}
	if err := setRecipeInstructionSections(ctx, q, recipe.ID, req.InstructionSections); err != nil {
		return nil, err
	}
	if err := setRecipeTags(ctx, q, recipe.ID, params.SpaceSlug, req.Tags, now); err != nil {
		return nil, err
	}
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityRecipe,
		EntityID:   types.FormatRecipeID(recipe.ID),
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "name", To: new(recipe.Name)},
		},
	}, now); err != nil {
		return nil, err
	}
	result, err := h.fetchRecipe(ctx, q, recipe.ID, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) SpaceRecipesList(ctx context.Context, params apigen.SpaceRecipesListParams) (*apigen.RecipePage, error) {
	if err := h.requireScope(ctx, "recipes:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	cursor, err := decodeRecipeListCursor(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	limit := clampLimit(params.Limit)
	q := dbgen.New(h.Pool)
	rows, err := q.ListRecipesBySpace(ctx, dbgen.ListRecipesBySpaceParams{
		SpaceSlug:       params.SpaceSlug,
		CursorID:        cursor.ID,
		CursorUpdatedAt: cursor.UpdatedAt,
		Lim:             limit + 1,
	})
	if err != nil {
		return nil, err
	}
	items, nextCursor, err := paginate(rows, limit, func(rows []dbgen.Recipe) ([]apigen.RecipeSummary, error) {
		return h.enrichRecipeSummaries(ctx, q, rows)
	}, encodeRecipeListCursor)
	if err != nil {
		return nil, err
	}
	return &apigen.RecipePage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) SpaceRecipesRead(ctx context.Context, params apigen.SpaceRecipesReadParams) (*apigen.Recipe, error) {
	if err := h.requireScope(ctx, "recipes:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	id, err := types.ParseRecipeID(params.RecipeId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	return h.fetchRecipe(ctx, dbgen.New(h.Pool), id, params.SpaceSlug)
}

func parseSearchRecipeID(query string) (int64, error) {
	if !exactRecipeIDPattern.MatchString(query) {
		return 0, nil
	}
	if strings.HasPrefix(query, "R") {
		return types.ParseRecipeID(query)
	}
	return types.ParseRecipeID("R" + query)
}

func (h *Handler) RecipesSearch(ctx context.Context, params apigen.RecipesSearchParams) (*apigen.RecipeSearchResultList, error) {
	if err := h.requireScope(ctx, "recipes:read"); err != nil {
		return nil, err
	}
	user := auth.UserFromContext(ctx)
	if user == nil {
		return nil, forbidden("authentication required")
	}
	queryText := strings.TrimSpace(params.Q)
	if queryText == "" {
		return nil, badRequest("query must not be empty")
	}
	exactID, err := parseSearchRecipeID(queryText)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	q := dbgen.New(h.Pool)
	rows, err := q.SearchVisibleRecipes(ctx, dbgen.SearchVisibleRecipesParams{
		ViewerUserID:  user.ID,
		SpaceSlug:     params.SpaceSlug.Or(""),
		ExactRecipeID: exactID,
		QueryText:     queryText,
		Lim:           clampLimit(params.Limit),
	})
	if err != nil {
		return nil, err
	}
	recipes := make([]dbgen.Recipe, len(rows))
	for i, row := range rows {
		recipes[i] = recipeFromSearchRow(row)
	}
	items, err := h.enrichRecipeSummaries(ctx, q, recipes)
	if err != nil {
		return nil, err
	}
	return &apigen.RecipeSearchResultList{Items: items}, nil
}

func mergeRecipeYield(update apigen.OptNilRecipeYield, existingAmount pgtype.Numeric, existingUnit pgtype.Text) (pgtype.Numeric, pgtype.Text, error) {
	if !update.IsSet() {
		return existingAmount, existingUnit, nil
	}
	if update.IsNull() {
		return pgtype.Numeric{}, pgtype.Text{}, nil
	}
	return recipeYieldToDB(update.Value)
}

func mergeOptionalInt32(update apigen.OptNilInt32, existing pgtype.Int4) pgtype.Int4 {
	if !update.IsSet() {
		return existing
	}
	if update.IsNull() {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: update.Value, Valid: true}
}

func (h *Handler) SpaceRecipesUpdate(ctx context.Context, req *apigen.RecipeUpdate, params apigen.SpaceRecipesUpdateParams) (*apigen.Recipe, error) {
	if err := h.requireScope(ctx, "recipes:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	id, err := types.ParseRecipeID(params.RecipeId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	if req.Name.IsSet() {
		if err := validateRecipeName(req.Name.Value); err != nil {
			return nil, err
		}
	}
	if err := validateRecipeCollections(req.IngredientSections, req.InstructionSections); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)
	existing, err := q.GetRecipeForUpdate(ctx, dbgen.GetRecipeForUpdateParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return nil, err
	}

	yieldAmount, yieldUnit, err := mergeRecipeYield(req.Yield, existing.YieldAmount, existing.YieldUnit)
	if err != nil {
		return nil, err
	}
	name := req.Name.Or(existing.Name)
	description := req.Description.Or(existing.Description)
	prepMinutes := mergeOptionalInt32(req.PrepMinutes, existing.PrepMinutes)
	cookMinutes := mergeOptionalInt32(req.CookMinutes, existing.CookMinutes)
	source := req.Source.Or(existing.Source)
	sourceURL := optNilStringToDB(req.SourceUrl, existing.SourceUrl)
	now := time.Now()
	if _, err := q.UpdateRecipe(ctx, dbgen.UpdateRecipeParams{
		Name:        name,
		Description: description,
		YieldAmount: yieldAmount,
		YieldUnit:   yieldUnit,
		PrepMinutes: prepMinutes,
		CookMinutes: cookMinutes,
		Source:      source,
		SourceUrl:   sourceURL,
		UpdatedAt:   types.Timestamptz(now),
		ID:          id,
		SpaceSlug:   params.SpaceSlug,
	}); err != nil {
		return nil, err
	}

	details := make([]activitylog.Detail, 0)
	if name != existing.Name {
		details = append(details, activitylog.Detail{Field: "name", From: new(existing.Name), To: new(name)})
	}
	if description != existing.Description {
		details = append(details, activitylog.Detail{Field: "description", From: new(existing.Description), To: new(description)})
	}
	if source != existing.Source {
		details = append(details, activitylog.Detail{Field: "source", From: new(existing.Source), To: new(source)})
	}
	if req.Yield.IsSet() {
		details = append(details, activitylog.Detail{Field: "yield", To: new("updated")})
	}
	if req.PrepMinutes.IsSet() {
		details = append(details, activitylog.Detail{Field: "prep_minutes", To: new("updated")})
	}
	if req.CookMinutes.IsSet() {
		details = append(details, activitylog.Detail{Field: "cook_minutes", To: new("updated")})
	}
	if req.SourceUrl.IsSet() {
		details = append(details, activitylog.Detail{Field: "source_url", To: new("updated")})
	}
	if req.IngredientSections != nil {
		if err := setRecipeIngredientSections(ctx, q, id, req.IngredientSections); err != nil {
			return nil, err
		}
		details = append(details, activitylog.Detail{Field: "ingredients", To: new("updated")})
	}
	if req.InstructionSections != nil {
		if err := setRecipeInstructionSections(ctx, q, id, req.InstructionSections); err != nil {
			return nil, err
		}
		details = append(details, activitylog.Detail{Field: "instructions", To: new("updated")})
	}
	if req.Tags != nil {
		oldTags, err := q.ListTagNamesByRecipe(ctx, id)
		if err != nil {
			return nil, err
		}
		foldedOld := make([]string, len(oldTags))
		for i, tag := range oldTags {
			foldedOld[i] = tagname.Fold(tag)
		}
		foldedNew := make([]string, len(req.Tags))
		for i, tag := range req.Tags {
			foldedNew[i] = tagname.Fold(tag)
		}
		details = append(details, diffStringList("tag", foldedOld, foldedNew)...)
		if err := setRecipeTags(ctx, q, id, params.SpaceSlug, req.Tags, now); err != nil {
			return nil, err
		}
	}
	if len(details) > 0 {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityRecipe,
			EntityID:   types.FormatRecipeID(id),
			Action:     activitylog.ActionUpdated,
			Details:    details,
		}, now); err != nil {
			return nil, err
		}
	}
	result, err := h.fetchRecipe(ctx, q, id, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) SpaceRecipesDelete(ctx context.Context, params apigen.SpaceRecipesDeleteParams) error {
	if err := h.requireScope(ctx, "recipes:write"); err != nil {
		return err
	}
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return err
	}
	id, err := types.ParseRecipeID(params.RecipeId)
	if err != nil {
		return badRequest(err.Error())
	}
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)
	existing, err := q.GetRecipe(ctx, dbgen.GetRecipeParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return err
	}
	result, err := q.DeleteRecipe(ctx, dbgen.DeleteRecipeParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return err
	}
	if err := checkDeleted(result); err != nil {
		return err
	}
	now := time.Now()
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityRecipe,
		EntityID:   types.FormatRecipeID(id),
		Action:     activitylog.ActionDeleted,
		Details: []activitylog.Detail{
			{Field: "name", From: new(existing.Name)},
		},
	}, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
