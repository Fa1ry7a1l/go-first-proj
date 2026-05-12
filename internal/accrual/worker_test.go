package accrual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

func TestWorkerProcessesOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"order":"12345678903","status":"PROCESSED","accrual":10.5}`))
	}))
	defer server.Close()

	storage := &workerFakeStorage{
		orders: []domain.Order{{Number: "12345678903", Status: domain.OrderStatusNew}},
	}
	worker := NewWorker(storage, NewClient(server.URL))

	worker.process(context.Background())

	if len(storage.updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(storage.updates))
	}
	if storage.updates[0].status != domain.OrderStatusProcessed {
		t.Fatalf("status = %s, want %s", storage.updates[0].status, domain.OrderStatusProcessed)
	}
	if storage.updates[0].accrual == nil || *storage.updates[0].accrual != 1050 {
		t.Fatalf("accrual = %v, want 1050", storage.updates[0].accrual)
	}
}

func TestWorkerMarksNoContentAsProcessing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	storage := &workerFakeStorage{
		orders: []domain.Order{{Number: "12345678903", Status: domain.OrderStatusNew}},
	}
	worker := NewWorker(storage, NewClient(server.URL))

	worker.process(context.Background())

	if len(storage.updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(storage.updates))
	}
	if storage.updates[0].status != domain.OrderStatusProcessing {
		t.Fatalf("status = %s, want %s", storage.updates[0].status, domain.OrderStatusProcessing)
	}
}

type workerFakeStorage struct {
	orders  []domain.Order
	updates []workerFakeUpdate
}

type workerFakeUpdate struct {
	number  string
	status  domain.OrderStatus
	accrual *domain.Points
}

func (s *workerFakeStorage) CreateOrder(_ context.Context, _ domain.Order) error {
	return nil
}

func (s *workerFakeStorage) GetOrderByNumber(_ context.Context, _ string) (domain.Order, error) {
	return domain.Order{}, domain.ErrOrderNotFound
}

func (s *workerFakeStorage) ListUserOrders(_ context.Context, _ int64) ([]domain.Order, error) {
	return nil, nil
}

func (s *workerFakeStorage) ListPendingOrders(_ context.Context, _ int) ([]domain.Order, error) {
	return s.orders, nil
}

func (s *workerFakeStorage) UpdateOrderAccrual(_ context.Context, number string, status domain.OrderStatus, accrual *domain.Points) error {
	s.updates = append(s.updates, workerFakeUpdate{
		number:  number,
		status:  status,
		accrual: accrual,
	})
	return nil
}
