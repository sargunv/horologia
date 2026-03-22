-- name: CreateSpaceMember :one
INSERT INTO space_members (space_slug, user_id, role, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetSpaceMember :one
SELECT * FROM space_members
WHERE space_slug = ? AND user_id = ?;

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
WHERE sm.space_slug = ? AND sm.user_id > ?
ORDER BY sm.user_id ASC
LIMIT ?;

-- name: ListSpacesByUser :many
SELECT s.* FROM spaces s
JOIN space_members sm ON sm.space_slug = s.slug
WHERE sm.user_id = ? AND s.slug > ?
ORDER BY s.slug ASC
LIMIT ?;

-- name: UpdateSpaceMemberRole :one
UPDATE space_members
SET role = ?
WHERE space_slug = ? AND user_id = ?
RETURNING *;

-- name: CountSpaceAdmins :one
SELECT COUNT(*) FROM space_members
WHERE space_slug = ? AND role = 'admin';

-- name: DeleteSpaceMember :execresult
DELETE FROM space_members WHERE space_slug = ? AND user_id = ?;
