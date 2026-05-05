package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

// CreateOrder сохраняет новый заказ пользователя в PostgreSQL.
func (s *Storage) CreateOrder(ctx context.Context, order domain.Order) error {
	const query = `
		INSERT INTO orders (number, user_id, status)
		VALUES ($1, $2, $3)
	`

	_, err := s.pool.Exec(ctx, query, order.Number, order.UserID, order.Status)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return domain.ErrOrderUploadedByAnotherUser
	}

	return fmt.Errorf("create order: %w", err)
}

// GetOrderByNumber возвращает заказ по его номеру.
func (s *Storage) GetOrderByNumber(ctx context.Context, number string) (domain.Order, error) {
	const query = `
		SELECT id, number, user_id, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE number = $1
	`

	order, err := scanOrder(s.pool.QueryRow(ctx, query, number))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order by number: %w", err)
	}

	return order, nil
}

// ListUserOrders возвращает заказы пользователя в порядке от новых к старым.
func (s *Storage) ListUserOrders(ctx context.Context, userID int64) ([]domain.Order, error) {
	const query = `
		SELECT id, number, user_id, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user orders: %w", err)
	}

	return orders, nil
}

// ListPendingOrders возвращает заказы, которые нужно проверить во внешней системе начислений.
func (s *Storage) ListPendingOrders(ctx context.Context, limit int) ([]domain.Order, error) {
	const query = `
		SELECT id, number, user_id, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY uploaded_at ASC
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, domain.OrderStatusNew, domain.OrderStatusProcessing, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending orders: %w", err)
	}

	return orders, nil
}

// UpdateOrderAccrual обновляет статус заказа и начисление, пока заказ не стал финальным.
func (s *Storage) UpdateOrderAccrual(ctx context.Context, number string, status domain.OrderStatus, accrual *domain.Points) error {
	const query = `
		UPDATE orders
		SET status = $2,
		    accrual = $3,
		    updated_at = NOW()
		WHERE number = $1
		  AND status IN ($4, $5)
	`

	var accrualValue *int64
	if accrual != nil {
		value := int64(*accrual)
		accrualValue = &value
	}

	_, err := s.pool.Exec(
		ctx,
		query,
		number,
		status,
		accrualValue,
		domain.OrderStatusNew,
		domain.OrderStatusProcessing,
	)
	if err != nil {
		return fmt.Errorf("update order accrual: %w", err)
	}
	return nil
}

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrder(scanner orderScanner) (domain.Order, error) {
	var order domain.Order
	var accrual *int64
	if err := scanner.Scan(
		&order.ID,
		&order.Number,
		&order.UserID,
		&order.Status,
		&accrual,
		&order.UploadedAt,
		&order.UpdatedAt,
	); err != nil {
		return domain.Order{}, err
	}
	if accrual != nil {
		points := domain.Points(*accrual)
		order.Accrual = &points
	}

	return order, nil
}
