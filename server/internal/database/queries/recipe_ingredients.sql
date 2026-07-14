-- name: InsertRecipeIngredientSection :one
INSERT INTO recipe_ingredient_sections (recipe_id, title, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: InsertRecipeIngredient :exec
INSERT INTO recipe_ingredients (
    section_id,
    position,
    quantity,
    quantity_max,
    unit,
    item,
    preparation,
    optional
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListRecipeIngredientRows :many
SELECT
    ris.id AS section_id,
    ris.title AS section_title,
    ris.position AS section_position,
    ri.id AS ingredient_id,
    ri.position AS ingredient_position,
    ri.quantity,
    ri.quantity_max,
    ri.unit,
    ri.item,
    ri.preparation,
    ri.optional
FROM recipe_ingredient_sections ris
LEFT JOIN recipe_ingredients ri ON ri.section_id = ris.id
WHERE ris.recipe_id = $1
ORDER BY ris.position, ri.position;

-- name: DeleteRecipeIngredientSections :exec
DELETE FROM recipe_ingredient_sections
WHERE recipe_id = $1;
