-- name: CreateDomain :one
INSERT INTO domains (
  id,
  domain,
  app_id,
  tls_enabled
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: GetDomain :one
SELECT * FROM domains
WHERE id = ?
LIMIT 1;

-- name: GetDomainByName :one
SELECT * FROM domains
WHERE domain = ?
LIMIT 1;

-- name: ListDomainsByApp :many
SELECT * FROM domains
WHERE app_id = ?
ORDER BY created_at DESC;

-- name: ListAllDomains :many
SELECT * FROM domains
ORDER BY created_at DESC;

-- name: UpdateDomain :exec
UPDATE domains
SET
  domain = ?,
  tls_enabled = ?
WHERE id = ?;

-- name: DeleteDomain :exec
DELETE FROM domains
WHERE id = ?;

-- name: GetActiveDomains :many
SELECT
  d.id,
  d.domain,
  d.app_id,
  d.tls_enabled,
  d.created_at
FROM domains d
INNER JOIN deployments dep ON dep.app_id = d.app_id
WHERE dep.status = 'active';
