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
UPDATE space_members
SET role = $1
WHERE space_slug = $2 AND user_id = $3
RETURNING *;

-- name: CountSpaceAdmins :one
SELECT COUNT(*) FROM space_members
WHERE space_slug = $1 AND role = 'admin';

-- name: ListSpaceMemberUserIDs :many
SELECT user_id FROM space_members
WHERE space_slug = $1
ORDER BY user_id ASC;

-- name: DeleteSpaceMember :execresult
DELETE FROM space_members WHERE space_slug = $1 AND user_id = $2;
