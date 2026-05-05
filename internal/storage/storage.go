// Package storage объявляет интерфейсы хранилища, от которых зависят сервисы.
package storage

import (
	"context"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

// UserStorage описывает операции чтения и записи пользователей.
type UserStorage interface {
	// CreateUser сохраняет нового пользователя.
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)

	// GetUserByLogin возвращает пользователя по логину.
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
}

// OrderStorage описывает операции чтения и записи заказов пользователя.
type OrderStorage interface {
	// CreateOrder сохраняет новый заказ пользователя.
	CreateOrder(ctx context.Context, order domain.Order) error

	// GetOrderByNumber возвращает заказ по его номеру.
	GetOrderByNumber(ctx context.Context, number string) (domain.Order, error)

	// ListUserOrders возвращает заказы пользователя в порядке от новых к старым.
	ListUserOrders(ctx context.Context, userID int64) ([]domain.Order, error)
}

// BalanceStorage описывает операции чтения баланса и записи списаний.
type BalanceStorage interface {
	// GetBalance возвращает текущий баланс пользователя.
	GetBalance(ctx context.Context, userID int64) (domain.Balance, error)

	// CreateWithdrawal создает списание баллов, если доступного баланса достаточно.
	CreateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error

	// ListWithdrawals возвращает списания пользователя в порядке от новых к старым.
	ListWithdrawals(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
}
