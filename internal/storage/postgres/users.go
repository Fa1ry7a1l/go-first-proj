package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateUser сохраняет нового пользователя в PostgreSQL.
func (s *Storage) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	const query = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id, login, password_hash, created_at
	`

	created, err := scanUser(s.pool.QueryRow(ctx, query, user.Login, user.PasswordHash))
	if err == nil {
		return created, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return domain.User{}, domain.ErrUserAlreadyExists
	}

	return domain.User{}, fmt.Errorf("create user: %w", err)
}

// GetUserByLogin возвращает пользователя по логину.
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	const query = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`

	user, err := scanUser(s.pool.QueryRow(ctx, query, login))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by login: %w", err)
	}

	return user, nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (domain.User, error) {
	var user domain.User
	if err := scanner.Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	); err != nil {
		return domain.User{}, err
	}

	return user, nil
}
