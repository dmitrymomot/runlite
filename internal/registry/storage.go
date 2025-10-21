package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrDomainNotFound     = errors.New("registry.domain.not_found")
	ErrDomainExists       = errors.New("registry.domain.exists")
	ErrDomainInvalid      = errors.New("registry.domain.invalid")
	ErrDatabaseConnection = errors.New("registry.database.connection")
)

type Domain struct {
	Domain    string
	CreatedAt int64
	Verified  bool
}

type Storage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewStorage(ctx context.Context, logger *slog.Logger, dbPath string) (*Storage, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}

	s := &Storage{
		db:     db,
		logger: logger,
	}

	if err := s.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("domain registry storage initialized", "path", dbPath)

	return s, nil
}

func (s *Storage) initSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS domains (
			domain TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL,
			verified BOOLEAN NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_domains_verified ON domains(verified);
		CREATE INDEX IF NOT EXISTS idx_domains_created_at ON domains(created_at);
	`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return ErrDomainInvalid
	}

	stmt, err := s.db.PrepareContext(ctx, "INSERT INTO domains (domain, created_at, verified) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	createdAt := time.Now().Unix()
	if _, err := stmt.ExecContext(ctx, domain, createdAt, false); err != nil {
		if err.Error() == "UNIQUE constraint failed: domains.domain" {
			return ErrDomainExists
		}
		return fmt.Errorf("failed to add domain: %w", err)
	}

	s.logger.Info("domain added", "domain", domain, "created_at", createdAt)

	return nil
}

func (s *Storage) RemoveDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return ErrDomainInvalid
	}

	stmt, err := s.db.PrepareContext(ctx, "DELETE FROM domains WHERE domain = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to remove domain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrDomainNotFound
	}

	s.logger.Info("domain removed", "domain", domain)

	return nil
}

func (s *Storage) GetDomain(ctx context.Context, domain string) (Domain, error) {
	if domain == "" {
		return Domain{}, ErrDomainInvalid
	}

	stmt, err := s.db.PrepareContext(ctx, "SELECT domain, created_at, verified FROM domains WHERE domain = ?")
	if err != nil {
		return Domain{}, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var d Domain
	if err := stmt.QueryRowContext(ctx, domain).Scan(&d.Domain, &d.CreatedAt, &d.Verified); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Domain{}, ErrDomainNotFound
		}
		return Domain{}, fmt.Errorf("failed to get domain: %w", err)
	}

	return d, nil
}

func (s *Storage) ListDomains(ctx context.Context) ([]Domain, error) {
	stmt, err := s.db.PrepareContext(ctx, "SELECT domain, created_at, verified FROM domains ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.Domain, &d.CreatedAt, &d.Verified); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domains: %w", err)
	}

	return domains, nil
}

func (s *Storage) VerifyDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return ErrDomainInvalid
	}

	stmt, err := s.db.PrepareContext(ctx, "UPDATE domains SET verified = ? WHERE domain = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, true, domain)
	if err != nil {
		return fmt.Errorf("failed to verify domain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrDomainNotFound
	}

	s.logger.Info("domain verified", "domain", domain)

	return nil
}

func (s *Storage) DomainExists(ctx context.Context, domain string) (bool, error) {
	if domain == "" {
		return false, ErrDomainInvalid
	}

	stmt, err := s.db.PrepareContext(ctx, "SELECT 1 FROM domains WHERE domain = ?")
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var exists int
	err = stmt.QueryRowContext(ctx, domain).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check domain existence: %w", err)
	}

	return true, nil
}
