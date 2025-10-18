-- +goose Up
-- +goose StatementBegin
CREATE TABLE deployment_logs (
  id TEXT PRIMARY KEY,
  deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  event TEXT NOT NULL,
  message TEXT,
  error TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_deployment_logs_deployment_id ON deployment_logs(deployment_id);
CREATE INDEX idx_deployment_logs_created_at ON deployment_logs(created_at);
CREATE INDEX idx_deployment_logs_deployment_id_created_at ON deployment_logs(deployment_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_deployment_logs_deployment_id_created_at;
DROP INDEX IF EXISTS idx_deployment_logs_created_at;
DROP INDEX IF EXISTS idx_deployment_logs_deployment_id;
DROP TABLE IF EXISTS deployment_logs;
-- +goose StatementEnd
