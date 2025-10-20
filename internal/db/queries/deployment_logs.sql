-- name: CreateDeploymentLog :exec
INSERT INTO deployment_logs (
  id,
  deployment_id,
  event,
  message,
  error
) VALUES (
  ?, ?, ?, ?, ?
);

-- name: GetDeploymentLogs :many
SELECT * FROM deployment_logs
WHERE deployment_id = ?
ORDER BY created_at ASC;

-- name: GetRecentDeploymentLogs :many
SELECT * FROM deployment_logs
WHERE deployment_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: DeleteDeploymentLogs :exec
DELETE FROM deployment_logs
WHERE deployment_id = ?;
