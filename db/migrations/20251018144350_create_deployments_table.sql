-- +goose Up
-- +goose StatementBegin
CREATE TABLE deployments (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  commit_hash TEXT NOT NULL,
  port INTEGER NOT NULL UNIQUE,
  host TEXT NOT NULL DEFAULT 'localhost',
  status TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  activated_at DATETIME,
  stopped_at DATETIME
);

CREATE INDEX idx_deployments_app_id ON deployments(app_id);
CREATE INDEX idx_deployments_app_id_status ON deployments(app_id, status);
CREATE INDEX idx_deployments_status ON deployments(status);

-- Ensure only ONE active deployment per app
CREATE UNIQUE INDEX idx_deployments_app_id_active ON deployments(app_id) WHERE status = 'active';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_deployments_app_id_active;
DROP INDEX IF EXISTS idx_deployments_status;
DROP INDEX IF EXISTS idx_deployments_app_id_status;
DROP INDEX IF EXISTS idx_deployments_app_id;
DROP TABLE IF EXISTS deployments;
-- +goose StatementEnd
