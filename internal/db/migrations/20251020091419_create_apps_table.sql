-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS apps (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    service_name TEXT NOT NULL,
    repo_url TEXT NOT NULL,
    branch TEXT NOT NULL DEFAULT 'main',
    static_path TEXT,
    health_check_path TEXT DEFAULT '/',
    health_check_timeout INTEGER NOT NULL DEFAULT 30000000000,  -- 30 seconds in nanoseconds
    health_check_interval INTEGER NOT NULL DEFAULT 10000000000, -- 10 seconds in nanoseconds
    drain_period INTEGER NOT NULL DEFAULT 30000000000,          -- 30 seconds in nanoseconds
    auto_deploy INTEGER NOT NULL DEFAULT 0,                     -- boolean: 0=false, 1=true
    github_install_id INTEGER,
    github_repo TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on name for faster lookups
CREATE INDEX IF NOT EXISTS idx_apps_name ON apps(name);

-- Create index on repo_url for webhook routing
CREATE INDEX IF NOT EXISTS idx_apps_repo_url ON apps(repo_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_apps_repo_url;
DROP INDEX IF EXISTS idx_apps_name;
DROP TABLE IF EXISTS apps;
-- +goose StatementEnd
