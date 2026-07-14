-- name: InsertRecipeInstructionSection :one
INSERT INTO recipe_instruction_sections (recipe_id, title, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: InsertRecipeStep :exec
INSERT INTO recipe_steps (section_id, position, body)
VALUES ($1, $2, $3);

-- name: ListRecipeInstructionRows :many
SELECT
    ris.id AS section_id,
    ris.title AS section_title,
    ris.position AS section_position,
    rs.id AS step_id,
    rs.position AS step_position,
    rs.body
FROM recipe_instruction_sections ris
LEFT JOIN recipe_steps rs ON rs.section_id = ris.id
WHERE ris.recipe_id = $1
ORDER BY ris.position, rs.position;

-- name: DeleteRecipeInstructionSections :exec
DELETE FROM recipe_instruction_sections
WHERE recipe_id = $1;
