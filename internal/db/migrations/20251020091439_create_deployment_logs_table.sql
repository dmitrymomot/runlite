-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS deployment_logs (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    event TEXT NOT NULL, -- build_start, build_output, build_complete, build_failed, health_check_start, health_check_pass, health_check_fail, traffic_switch, drain_start, drain_complete, deployment_failed, deployment_complete
    message TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Create index on deployment_id for faster log queries
CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id ON deployment_logs(deployment_id);

-- Create composite index on deployment_id and created_at for sorted queries
CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_created ON deployment_logs(deployment_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_deployment_logs_deployment_created;
DROP INDEX IF EXISTS idx_deployment_logs_deployment_id;
DROP TABLE IF EXISTS deployment_logs;
-- +goose StatementEnd
