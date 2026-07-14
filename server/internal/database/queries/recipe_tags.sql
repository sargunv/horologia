-- name: ListTagNamesByRecipe :many
SELECT tg.name FROM recipe_tags rt
JOIN tags tg ON tg.id = rt.tag_id
WHERE rt.recipe_id = $1
ORDER BY tg.name;

-- name: ListTagNamesByRecipes :many
SELECT rt.recipe_id, tg.name FROM recipe_tags rt
JOIN tags tg ON tg.id = rt.tag_id
WHERE rt.recipe_id = ANY(@recipe_ids::bigint[])
ORDER BY rt.recipe_id, tg.name;

-- name: InsertRecipeTag :exec
INSERT INTO recipe_tags (recipe_id, tag_id, space_slug, created_at)
VALUES ($1, $2, $3, $4);

-- name: DeleteRecipeTags :exec
DELETE FROM recipe_tags
WHERE recipe_id = $1;
