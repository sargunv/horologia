-- name: CreateSpaceMember :one
INSERT INTO space_members (space_slug, user_id, role, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSpaceMember :one
SELECT * FROM space_members
WHERE space_slug = $1 AND user_id = $2;

-- name: ListSpaceMembersBySpace :many
SELECT
    sm.space_slug,
    sm.user_id,
    sm.role,
    sm.created_at,
    u.name  AS user_name,
    u.email AS user_email
FROM space_members sm
JOIN users u ON u.id = sm.user_id
WHERE sm.space_slug = $1 AND sm.user_id > $2
ORDER BY sm.user_id ASC
LIMIT $3;

-- name: ListSpacesByUser :many
SELECT s.* FROM spaces s
JOIN space_members sm ON sm.space_slug = s.slug
WHERE sm.user_id = $1
ORDER BY s.slug ASC;

-- name: UpdateSpaceMemberRole :one
-- Atomically updates a member's role, refusing to demote the last admin.
-- Returns the updated row, or no rows if the member doesn't exist or the
-- update would leave the space with zero admins.
UPDATE space_members sm
SET role = $1
WHERE sm.space_slug = $2 AND sm.user_id = $3
  AND ($1::space_role = 'admin' OR sm.role != 'admin'
       OR (SELECT COUNT(*) FROM space_members WHERE space_slug = sm.space_slug AND role = 'admin') > 1)
RETURNING *;

-- name: ListSpaceMemberUserIDs :many
SELECT user_id FROM space_members
WHERE space_slug = $1
ORDER BY user_id ASC;

-- name: DeleteSpaceMember :execresult
-- Atomically deletes a member, refusing to delete the last admin.
-- Affects zero rows if the member is the sole admin.
DELETE FROM space_members sm
WHERE sm.space_slug = $1 AND sm.user_id = $2
  AND (sm.role != 'admin'
       OR (SELECT COUNT(*) FROM space_members WHERE space_slug = sm.space_slug AND role = 'admin') > 1);
