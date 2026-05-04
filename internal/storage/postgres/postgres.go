// Package postgres реализует интерфейсы хранилища поверх PostgreSQL.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Storage предоставляет доступ к данным, сохраненным в PostgreSQL.
type Storage struct {
	pool *pgxpool.Pool
}

// New открывает пул подключений к PostgreSQL и применяет встроенные миграции.
func New(ctx context.Context, databaseURI string) (*Storage, error) {
	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	storage := &Storage{pool: pool}
	if err := storage.applyMigrations(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return storage, nil
}

// Close освобождает все подключения PostgreSQL, открытые хранилищем.
func (s *Storage) Close() {
	s.pool.Close()
}

func (s *Storage) applyMigrations(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		query, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}
