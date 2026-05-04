// Package httpapi содержит маршрутизацию и HTTP-обработчики API Gophermart.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
)

const (
	mvpUserID int64 = 1
)

// Router обрабатывает HTTP-запросы к API Gophermart.
type Router struct {
	orders *service.OrderService
}

// NewRouter создает дерево HTTP-обработчиков сервиса.
func NewRouter(orders *service.OrderService) http.Handler {
	router := &Router{
		orders: orders,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handlePing)
	mux.HandleFunc("/api/user/orders", router.handleUserOrders)
	return mux
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (rt *Router) handleUserOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		rt.handleUploadOrder(w, r)
	case http.MethodGet:
		rt.handleListOrders(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *Router) handleUploadOrder(w http.ResponseWriter, r *http.Request) {
	if rt.orders == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = rt.orders.UploadOrder(r.Context(), mvpUserID, string(body))
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, domain.ErrOrderAlreadyUploadedByUser):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrOrderUploadedByAnotherUser):
		w.WriteHeader(http.StatusConflict)
	case errors.Is(err, domain.ErrOrderInvalidFormat):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, domain.ErrOrderInvalidNumber):
		w.WriteHeader(http.StatusUnprocessableEntity)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Router) handleListOrders(w http.ResponseWriter, r *http.Request) {
	if rt.orders == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orders, err := rt.orders.ListOrders(r.Context(), mvpUserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, orderResponse{
			Number:     order.Number,
			Status:     string(order.Status),
			Accrual:    pointsPtrToFloat64(order.Accrual),
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

type orderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func pointsPtrToFloat64(points *domain.Points) *float64 {
	if points == nil {
		return nil
	}
	value := points.Float64()
	return &value
}
