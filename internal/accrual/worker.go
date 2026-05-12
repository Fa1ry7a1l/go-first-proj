package accrual

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage"
)

const (
	defaultPollInterval = time.Second
	defaultBatchSize    = 10
)

// Worker периодически обновляет статусы заказов через внешнюю систему начислений.
type Worker struct {
	orders       storage.OrderStorage
	client       *Client
	pollInterval time.Duration
	batchSize    int
}

// NewWorker создает фоновый обработчик заказов.
func NewWorker(orders storage.OrderStorage, client *Client) *Worker {
	return &Worker{
		orders:       orders,
		client:       client,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
	}
}

// Run запускает цикл обработки и работает до отмены контекста.
func (w *Worker) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			delay := w.process(ctx)
			timer.Reset(delay)
		}
	}
}

func (w *Worker) process(ctx context.Context) time.Duration {
	orders, err := w.orders.ListPendingOrders(ctx, w.batchSize)
	if err != nil {
		slog.Error("не удалось получить заказы для проверки начислений", "error", err)
		return w.pollInterval
	}
	if len(orders) == 0 {
		return w.pollInterval
	}

	for _, order := range orders {
		result, err := w.client.FetchOrder(ctx, order.Number)
		switch {
		case err == nil:
			if err := w.orders.UpdateOrderAccrual(ctx, order.Number, result.Status, result.Accrual); err != nil {
				slog.Error("не удалось обновить начисление заказа", "order", order.Number, "error", err)
			}
		case errors.Is(err, ErrNoAccrualData):
			if order.Status == domain.OrderStatusNew {
				if err := w.orders.UpdateOrderAccrual(ctx, order.Number, domain.OrderStatusProcessing, nil); err != nil {
					slog.Error("не удалось перевести заказ в обработку", "order", order.Number, "error", err)
				}
			}
		case errors.Is(err, ErrRateLimited):
			if result.RetryAfter > 0 {
				return result.RetryAfter
			}
			return time.Minute
		default:
			slog.Error("не удалось получить начисление заказа", "order", order.Number, "error", err)
		}
	}

	return w.pollInterval
}
