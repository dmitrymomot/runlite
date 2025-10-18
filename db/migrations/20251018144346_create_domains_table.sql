-- +goose Up
-- +goose StatementBegin
CREATE TABLE domains (
  id TEXT PRIMARY KEY,
  domain TEXT NOT NULL UNIQUE,
  app_id TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  tls_enabled BOOLEAN NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_domains_app_id ON domains(app_id);
CREATE INDEX idx_domains_domain ON domains(domain);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_domains_domain;
DROP INDEX IF EXISTS idx_domains_app_id;
DROP TABLE IF EXISTS domains;
-- +goose StatementEnd
