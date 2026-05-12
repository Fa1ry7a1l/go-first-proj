// Package accrual содержит клиент и фоновую обработку внешней системы начислений.
package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

// Ошибки клиента внешней системы начислений.
var (
	// ErrNoAccrualData означает, что внешняя система пока не знает переданный заказ.
	ErrNoAccrualData = errors.New("accrual data not found")

	// ErrRateLimited означает, что внешняя система временно ограничила частоту запросов.
	ErrRateLimited = errors.New("accrual rate limited")
)

// Client получает статусы заказов из внешней системы начислений.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создает HTTP-клиент внешней системы начислений.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Result содержит результат запроса начислений по заказу.
type Result struct {
	// Order содержит номер заказа, который вернула внешняя система.
	Order string

	// Status содержит локальный статус обработки заказа.
	Status domain.OrderStatus

	// Accrual содержит начисленные баллы в минимальных целых единицах.
	Accrual *domain.Points

	// RetryAfter содержит паузу, которую нужно выдержать после ответа 429.
	RetryAfter time.Duration
}

// FetchOrder получает статус и начисления заказа из внешней системы.
func (c *Client) FetchOrder(ctx context.Context, number string) (Result, error) {
	url := c.baseURL + "/api/orders/" + number
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, fmt.Errorf("create accrual request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("fetch accrual order: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		var body responseBody
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			return Result{}, fmt.Errorf("decode accrual response: %w", err)
		}

		result := Result{
			Order:  body.Order,
			Status: mapAccrualStatus(body.Status),
		}
		if body.Accrual != nil {
			points := domain.PointsFromFloat64(*body.Accrual)
			result.Accrual = &points
		}
		return result, nil
	case http.StatusNoContent:
		return Result{}, ErrNoAccrualData
	case http.StatusTooManyRequests:
		return Result{RetryAfter: retryAfter(response.Header.Get("Retry-After"))}, ErrRateLimited
	default:
		return Result{}, fmt.Errorf("unexpected accrual status: %d", response.StatusCode)
	}
}

type responseBody struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

func mapAccrualStatus(status string) domain.OrderStatus {
	switch status {
	case "REGISTERED", "PROCESSING":
		return domain.OrderStatusProcessing
	case "INVALID":
		return domain.OrderStatusInvalid
	case "PROCESSED":
		return domain.OrderStatusProcessed
	default:
		return domain.OrderStatusProcessing
	}
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Minute
	}
	return time.Duration(seconds) * time.Second
}
