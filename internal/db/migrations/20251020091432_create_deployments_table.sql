-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    commit_hash TEXT NOT NULL,
    port INTEGER NOT NULL,
    host TEXT,
    binary_path TEXT,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, building, starting, healthy, unhealthy, stopped, failed, active, cancelling, cancelled
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at TIMESTAMP,
    stopped_at TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

-- Create index on app_id for faster queries
CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON deployments(app_id);

-- Create composite index on app_id and status for active deployment queries
CREATE INDEX IF NOT EXISTS idx_deployments_app_status ON deployments(app_id, status);

-- Create index on status for queue processing
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);

-- Create index on created_at for sorting
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_deployments_created_at;
DROP INDEX IF EXISTS idx_deployments_status;
DROP INDEX IF EXISTS idx_deployments_app_status;
DROP INDEX IF EXISTS idx_deployments_app_id;
DROP TABLE IF EXISTS deployments;
-- +goose StatementEnd
