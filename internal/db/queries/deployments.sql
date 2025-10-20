-- name: CreateDeployment :one
INSERT INTO deployments (
  id,
  app_id,
  commit_hash,
  port,
  host,
  status
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetDeployment :one
SELECT * FROM deployments
WHERE id = ?
LIMIT 1;

-- name: ListDeploymentsByApp :many
SELECT * FROM deployments
WHERE app_id = ?
ORDER BY created_at DESC;

-- name: ListDeploymentsByStatus :many
SELECT * FROM deployments
WHERE status = ?
ORDER BY created_at ASC;

-- name: GetActiveDeploymentByApp :one
SELECT * FROM deployments
WHERE app_id = ? AND status = 'active'
LIMIT 1;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = ?
WHERE id = ?;

-- name: UpdateDeploymentStatusWithActivatedAt :exec
UPDATE deployments
SET
  status = ?,
  activated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateDeploymentStatusWithStoppedAt :exec
UPDATE deployments
SET
  status = ?,
  stopped_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetDeploymentsNeedingWork :many
SELECT * FROM deployments
WHERE status IN (
  'pending',
  'building',
  'starting',
  'health_checking',
  'ready',
  'activating',
  'cancelling'
)
ORDER BY created_at ASC;

-- name: GetStaleDeployments :many
SELECT * FROM deployments
WHERE status IN (
  'building',
  'starting',
  'health_checking',
  'activating'
)
AND created_at < ?
ORDER BY created_at ASC;

-- name: DeleteDeployment :exec
DELETE FROM deployments
WHERE id = ?;

-- name: GetDeploymentsByAppAndStatuses :many
SELECT * FROM deployments
WHERE app_id = ? AND status IN (sqlc.slice('statuses'))
ORDER BY created_at DESC;
