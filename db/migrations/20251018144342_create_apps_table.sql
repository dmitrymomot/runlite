-- +goose Up
-- +goose StatementBegin
CREATE TABLE apps (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  health_check_path TEXT NOT NULL DEFAULT '/health',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_apps_name ON apps(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_apps_name;
DROP TABLE IF EXISTS apps;
-- +goose StatementEnd
