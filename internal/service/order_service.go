// Package service содержит бизнес-логику накопительной системы Gophermart.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/luhn"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage"
)

// OrderService выполняет бизнес-операции с заказами пользователей.
type OrderService struct {
	orders storage.OrderStorage
}

// NewOrderService создает сервис заказов поверх переданного хранилища.
func NewOrderService(orders storage.OrderStorage) *OrderService {
	return &OrderService{
		orders: orders,
	}
}

// UploadOrder проверяет номер заказа, учитывает конфликты владения и сохраняет
// новый заказ пользователя со статусом NEW.
func (s *OrderService) UploadOrder(ctx context.Context, userID int64, number string) error {
	number = strings.TrimSpace(number)
	if !digitsOnly(number) {
		return domain.ErrOrderInvalidFormat
	}
	if !luhn.Valid(number) {
		return domain.ErrOrderInvalidNumber
	}

	existing, err := s.orders.GetOrderByNumber(ctx, number)
	if err == nil {
		if existing.UserID == userID {
			return domain.ErrOrderAlreadyUploadedByUser
		}
		return domain.ErrOrderUploadedByAnotherUser
	}
	if !errors.Is(err, domain.ErrOrderNotFound) {
		return err
	}

	err = s.orders.CreateOrder(ctx, domain.Order{
		UserID: userID,
		Number: number,
		Status: domain.OrderStatusNew,
	})
	if errors.Is(err, domain.ErrOrderUploadedByAnotherUser) {
		existing, getErr := s.orders.GetOrderByNumber(ctx, number)
		if getErr != nil {
			return err
		}
		if existing.UserID == userID {
			return domain.ErrOrderAlreadyUploadedByUser
		}
	}
	return err
}

// ListOrders возвращает заказы пользователя в порядке от самых новых к самым старым.
func (s *OrderService) ListOrders(ctx context.Context, userID int64) ([]domain.Order, error) {
	return s.orders.ListUserOrders(ctx, userID)
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
