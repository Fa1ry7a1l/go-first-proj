package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/jackc/pgx/v5"
)

// GetBalance возвращает текущий баланс пользователя.
func (s *Storage) GetBalance(ctx context.Context, userID int64) (domain.Balance, error) {
	balance, err := s.getBalance(ctx, s.pool, userID)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

// CreateWithdrawal создает списание баллов в транзакции.
func (s *Storage) CreateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin withdrawal tx: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", withdrawal.UserID); err != nil {
		return fmt.Errorf("lock user balance: %w", err)
	}

	balance, err := s.getBalance(ctx, tx, withdrawal.UserID)
	if err != nil {
		return fmt.Errorf("get balance in tx: %w", err)
	}
	if balance.Current < withdrawal.Sum {
		return domain.ErrInsufficientFunds
	}

	const query = `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3)
	`
	if _, err := tx.Exec(ctx, query, withdrawal.UserID, withdrawal.OrderNumber, withdrawal.Sum); err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit withdrawal tx: %w", err)
	}
	return nil
}

// ListWithdrawals возвращает списания пользователя в порядке от новых к старым.
func (s *Storage) ListWithdrawals(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	const query = `
		SELECT id, user_id, order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []domain.Withdrawal
	for rows.Next() {
		withdrawal, err := scanWithdrawal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate withdrawals: %w", err)
	}

	return withdrawals, nil
}

type balanceQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type withdrawalScanner interface {
	Scan(dest ...any) error
}

func (s *Storage) getBalance(ctx context.Context, querier balanceQuerier, userID int64) (domain.Balance, error) {
	const query = `
		SELECT
			COALESCE((
				SELECT SUM(accrual)
				FROM orders
				WHERE user_id = $1
				  AND status = $2
				  AND accrual IS NOT NULL
			), 0) AS accrued,
			COALESCE((
				SELECT SUM(sum)
				FROM withdrawals
				WHERE user_id = $1
			), 0) AS withdrawn
	`

	var accrued int64
	var withdrawn int64
	if err := querier.QueryRow(ctx, query, userID, domain.OrderStatusProcessed).Scan(&accrued, &withdrawn); err != nil {
		return domain.Balance{}, err
	}

	return domain.Balance{
		Current:   domain.Points(accrued - withdrawn),
		Withdrawn: domain.Points(withdrawn),
	}, nil
}

func scanWithdrawal(scanner withdrawalScanner) (domain.Withdrawal, error) {
	var withdrawal domain.Withdrawal
	var sum int64
	if err := scanner.Scan(
		&withdrawal.ID,
		&withdrawal.UserID,
		&withdrawal.OrderNumber,
		&sum,
		&withdrawal.ProcessedAt,
	); err != nil {
		return domain.Withdrawal{}, err
	}
	withdrawal.Sum = domain.Points(sum)
	return withdrawal, nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return
	}
}
