package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

func TestOrderServiceUploadOrder(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		number    string
		existing  *domain.Order
		createErr error
		wantErr   error
	}{
		{name: "new order", number: "12345678903"},
		{name: "invalid format", number: "123abc", wantErr: domain.ErrOrderInvalidFormat},
		{name: "invalid number", number: "12345678904", wantErr: domain.ErrOrderInvalidNumber},
		{
			name:     "same user duplicate",
			number:   "12345678903",
			existing: &domain.Order{UserID: 1, Number: "12345678903"},
			wantErr:  domain.ErrOrderAlreadyUploadedByUser,
		},
		{
			name:     "another user duplicate",
			number:   "12345678903",
			existing: &domain.Order{UserID: 2, Number: "12345678903"},
			wantErr:  domain.ErrOrderUploadedByAnotherUser,
		},
		{
			name:      "storage create error",
			number:    "12345678903",
			createErr: errors.New("boom"),
			wantErr:   errors.New("boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := &fakeOrderStorage{
				existing:  tt.existing,
				createErr: tt.createErr,
			}
			service := NewOrderService(orders)

			err := service.UploadOrder(ctx, 1, tt.number)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("UploadOrder returned error: %v", err)
				}
				if len(orders.created) != 1 {
					t.Fatalf("created orders = %d, want 1", len(orders.created))
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr.Error() {
				t.Fatalf("UploadOrder error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

type fakeOrderStorage struct {
	existing  *domain.Order
	createErr error
	created   []domain.Order
}

func (s *fakeOrderStorage) CreateOrder(_ context.Context, order domain.Order) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, order)
	return nil
}

func (s *fakeOrderStorage) GetOrderByNumber(_ context.Context, _ string) (domain.Order, error) {
	if s.existing == nil {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return *s.existing, nil
}

func (s *fakeOrderStorage) ListUserOrders(_ context.Context, _ int64) ([]domain.Order, error) {
	return nil, nil
}

func (s *fakeOrderStorage) ListPendingOrders(_ context.Context, _ int) ([]domain.Order, error) {
	return nil, nil
}

func (s *fakeOrderStorage) UpdateOrderAccrual(_ context.Context, _ string, _ domain.OrderStatus, _ *domain.Points) error {
	return nil
}
