package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

func TestBalanceServiceWithdraw(t *testing.T) {
	storage := &fakeBalanceStorage{balance: domain.Balance{Current: 10000}}
	service := NewBalanceService(storage)

	err := service.Withdraw(context.Background(), 1, "12345678903", 2500)
	if err != nil {
		t.Fatalf("Withdraw returned error: %v", err)
	}
	if len(storage.withdrawals) != 1 {
		t.Fatalf("withdrawals = %d, want 1", len(storage.withdrawals))
	}
	if storage.withdrawals[0].Sum != 2500 {
		t.Fatalf("sum = %d, want 2500", storage.withdrawals[0].Sum)
	}
}

func TestBalanceServiceWithdrawRejectsInvalidOrder(t *testing.T) {
	service := NewBalanceService(&fakeBalanceStorage{})

	err := service.Withdraw(context.Background(), 1, "12345678904", 100)
	if !errors.Is(err, domain.ErrOrderInvalidNumber) {
		t.Fatalf("Withdraw error = %v, want %v", err, domain.ErrOrderInvalidNumber)
	}
}

func TestBalanceServiceWithdrawRejectsInvalidSum(t *testing.T) {
	service := NewBalanceService(&fakeBalanceStorage{})

	err := service.Withdraw(context.Background(), 1, "12345678903", 0)
	if !errors.Is(err, domain.ErrWithdrawalInvalidSum) {
		t.Fatalf("Withdraw error = %v, want %v", err, domain.ErrWithdrawalInvalidSum)
	}
}

func TestBalanceServiceWithdrawRejectsInsufficientFunds(t *testing.T) {
	service := NewBalanceService(&fakeBalanceStorage{balance: domain.Balance{Current: 100}})

	err := service.Withdraw(context.Background(), 1, "12345678903", 200)
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("Withdraw error = %v, want %v", err, domain.ErrInsufficientFunds)
	}
}

type fakeBalanceStorage struct {
	balance     domain.Balance
	withdrawals []domain.Withdrawal
}

func (s *fakeBalanceStorage) GetBalance(_ context.Context, _ int64) (domain.Balance, error) {
	return s.balance, nil
}

func (s *fakeBalanceStorage) CreateWithdrawal(_ context.Context, withdrawal domain.Withdrawal) error {
	if s.balance.Current < withdrawal.Sum {
		return domain.ErrInsufficientFunds
	}
	s.balance.Current -= withdrawal.Sum
	s.balance.Withdrawn += withdrawal.Sum
	s.withdrawals = append(s.withdrawals, withdrawal)
	return nil
}

func (s *fakeBalanceStorage) ListWithdrawals(_ context.Context, _ int64) ([]domain.Withdrawal, error) {
	return s.withdrawals, nil
}
