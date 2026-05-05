package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

func TestClientFetchOrderProcessed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"123","status":"PROCESSED","accrual":12.34}`))
	}))
	defer server.Close()

	result, err := NewClient(server.URL).FetchOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("FetchOrder returned error: %v", err)
	}
	if result.Status != domain.OrderStatusProcessed {
		t.Fatalf("status = %s, want %s", result.Status, domain.OrderStatusProcessed)
	}
	if result.Accrual == nil || *result.Accrual != 1234 {
		t.Fatalf("accrual = %v, want 1234", result.Accrual)
	}
}

func TestClientFetchOrderNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	_, err := NewClient(server.URL).FetchOrder(context.Background(), "123")
	if !errors.Is(err, ErrNoAccrualData) {
		t.Fatalf("FetchOrder error = %v, want %v", err, ErrNoAccrualData)
	}
}

func TestClientFetchOrderRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	result, err := NewClient(server.URL).FetchOrder(context.Background(), "123")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("FetchOrder error = %v, want %v", err, ErrRateLimited)
	}
	if result.RetryAfter != 2*time.Second {
		t.Fatalf("RetryAfter = %s, want 2s", result.RetryAfter)
	}
}
