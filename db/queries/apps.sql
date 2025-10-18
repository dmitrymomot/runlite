-- name: CreateApp :one
INSERT INTO apps (
  id,
  name,
  health_check_path
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: GetApp :one
SELECT * FROM apps
WHERE id = ?
LIMIT 1;

-- name: GetAppByName :one
SELECT * FROM apps
WHERE name = ?
LIMIT 1;

-- name: ListApps :many
SELECT * FROM apps
ORDER BY created_at DESC;

-- name: UpdateApp :exec
UPDATE apps
SET
  name = ?,
  health_check_path = ?,
  updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteApp :exec
DELETE FROM apps
WHERE id = ?;
