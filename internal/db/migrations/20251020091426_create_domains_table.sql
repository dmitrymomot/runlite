-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS domains (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,
    app_id TEXT NOT NULL,
    tls_enabled INTEGER NOT NULL DEFAULT 1, -- boolean: 0=false, 1=true
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

-- Create index on app_id for faster queries
CREATE INDEX IF NOT EXISTS idx_domains_app_id ON domains(app_id);

-- Create index on domain for faster lookups
CREATE INDEX IF NOT EXISTS idx_domains_domain ON domains(domain);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_domains_domain;
DROP INDEX IF EXISTS idx_domains_app_id;
DROP TABLE IF EXISTS domains;
-- +goose StatementEnd
