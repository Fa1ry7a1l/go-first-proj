// Package storage объявляет интерфейсы хранилища, от которых зависят сервисы.
package storage

import (
	"context"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

// OrderStorage описывает операции чтения и записи заказов пользователя.
type OrderStorage interface {
	// CreateOrder сохраняет новый заказ пользователя.
	CreateOrder(ctx context.Context, order domain.Order) error

	// GetOrderByNumber возвращает заказ по его номеру.
	GetOrderByNumber(ctx context.Context, number string) (domain.Order, error)

	// ListUserOrders возвращает заказы пользователя в порядке от новых к старым.
	ListUserOrders(ctx context.Context, userID int64) ([]domain.Order, error)
}
