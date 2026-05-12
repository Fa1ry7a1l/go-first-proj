package service

import (
	"context"
	"strings"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/luhn"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage"
)

// BalanceService выполняет бизнес-операции с балансом и списаниями пользователя.
type BalanceService struct {
	balances storage.BalanceStorage
}

// NewBalanceService создает сервис баланса поверх переданного хранилища.
func NewBalanceService(balances storage.BalanceStorage) *BalanceService {
	return &BalanceService{
		balances: balances,
	}
}

// GetBalance возвращает текущий баланс пользователя.
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (domain.Balance, error) {
	return s.balances.GetBalance(ctx, userID)
}

// Withdraw проверяет номер заказа и регистрирует списание баллов.
func (s *BalanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum domain.Points) error {
	orderNumber = strings.TrimSpace(orderNumber)
	if !digitsOnly(orderNumber) || !luhn.Valid(orderNumber) {
		return domain.ErrOrderInvalidNumber
	}
	if sum <= 0 {
		return domain.ErrWithdrawalInvalidSum
	}

	return s.balances.CreateWithdrawal(ctx, domain.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
	})
}

// ListWithdrawals возвращает историю списаний пользователя.
func (s *BalanceService) ListWithdrawals(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	return s.balances.ListWithdrawals(ctx, userID)
}
