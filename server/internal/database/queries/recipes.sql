-- name: CreateRecipe :one
INSERT INTO recipes (
    space_slug,
    name,
    description,
    yield_amount,
    yield_unit,
    prep_minutes,
    cook_minutes,
    source,
    source_url,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes
WHERE id = $1 AND space_slug = $2;

-- name: GetRecipeForUpdate :one
SELECT * FROM recipes
WHERE id = $1 AND space_slug = $2
FOR UPDATE;

-- name: ListRecipesBySpace :many
SELECT * FROM recipes
WHERE
    space_slug = @space_slug
    AND (
        @cursor_id::bigint = 0
        OR (updated_at, id) < (@cursor_updated_at::timestamptz, @cursor_id::bigint)
    )
ORDER BY updated_at DESC, id DESC
LIMIT @lim;

-- name: UpdateRecipe :one
UPDATE recipes
SET
    name = $1,
    description = $2,
    yield_amount = $3,
    yield_unit = $4,
    prep_minutes = $5,
    cook_minutes = $6,
    source = $7,
    source_url = $8,
    updated_at = $9
WHERE id = $10 AND space_slug = $11
RETURNING *;

-- name: DeleteRecipe :execresult
DELETE FROM recipes
WHERE id = $1 AND space_slug = $2;

-- name: SearchVisibleRecipes :many
SELECT
    vr.id,
    vr.space_slug,
    vr.name,
    vr.description,
    vr.yield_amount,
    vr.yield_unit,
    vr.prep_minutes,
    vr.cook_minutes,
    vr.source,
    vr.source_url,
    vr.created_at,
    vr.updated_at
FROM visible_recipes vr
WHERE
    vr.viewer_user_id = sqlc.arg(viewer_user_id)
    AND (sqlc.arg(space_slug)::text = '' OR vr.space_slug = sqlc.arg(space_slug)::text)
    AND (
        (sqlc.arg(exact_recipe_id)::bigint != 0 AND vr.id = sqlc.arg(exact_recipe_id)::bigint)
        OR (
            sqlc.arg(query_text)::text != ''
            AND (
                lower(vr.name) LIKE '%' || lower(sqlc.arg(query_text)::text) || '%'
                OR lower(vr.name) % lower(sqlc.arg(query_text)::text)
            )
        )
    )
ORDER BY
    CASE
        WHEN sqlc.arg(exact_recipe_id)::bigint != 0 AND vr.id = sqlc.arg(exact_recipe_id)::bigint THEN 0
        WHEN lower(vr.name) = lower(sqlc.arg(query_text)::text) THEN 1
        WHEN lower(vr.name) LIKE lower(sqlc.arg(query_text)::text) || '%' THEN 2
        WHEN lower(vr.name) LIKE '%' || lower(sqlc.arg(query_text)::text) || '%' THEN 3
        ELSE 4
    END ASC,
    similarity(lower(vr.name), lower(sqlc.arg(query_text)::text)) DESC,
    vr.updated_at DESC,
    vr.id DESC
LIMIT sqlc.arg(lim);
