// Package httpapi содержит маршрутизацию и HTTP-обработчики API Gophermart.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
)

const (
	authCookieName = "gophermart_auth"
)

// Router обрабатывает HTTP-запросы к API Gophermart.
type Router struct {
	users    *service.UserService
	orders   *service.OrderService
	balances *service.BalanceService
	tokens   *auth.TokenManager
}

// NewRouter создает дерево HTTP-обработчиков сервиса.
func NewRouter(
	users *service.UserService,
	orders *service.OrderService,
	balances *service.BalanceService,
	tokens *auth.TokenManager,
) http.Handler {
	router := &Router{
		users:    users,
		orders:   orders,
		balances: balances,
		tokens:   tokens,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handlePing)
	mux.HandleFunc("/api/user/register", router.handleRegister)
	mux.HandleFunc("/api/user/login", router.handleLogin)
	mux.HandleFunc("/api/user/orders", router.handleUserOrders)
	mux.HandleFunc("/api/user/balance", router.handleBalance)
	mux.HandleFunc("/api/user/balance/withdraw", router.handleWithdraw)
	mux.HandleFunc("/api/user/withdrawals", router.handleWithdrawals)
	return mux
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (rt *Router) handleUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodPost:
		rt.handleUploadOrder(w, r, userID)
	case http.MethodGet:
		rt.handleListOrders(w, r, userID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *Router) handleBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if rt.balances == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := rt.balances.GetBalance(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{
		Current:   balance.Current.Float64(),
		Withdrawn: balance.Withdrawn.Float64(),
	})
}

func (rt *Router) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if rt.balances == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var request withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := rt.balances.Withdraw(r.Context(), userID, request.Order, domain.PointsFromFloat64(request.Sum))
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrInsufficientFunds):
		w.WriteHeader(http.StatusPaymentRequired)
	case errors.Is(err, domain.ErrOrderInvalidNumber):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Is(err, domain.ErrWithdrawalInvalidSum):
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Router) handleWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if rt.balances == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	withdrawals, err := rt.balances.ListWithdrawals(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]withdrawalResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		response = append(response, withdrawalResponse{
			Order:       withdrawal.OrderNumber,
			Sum:         withdrawal.Sum.Float64(),
			ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (rt *Router) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if rt.users == nil || rt.tokens == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	request, ok := readCredentials(w, r)
	if !ok {
		return
	}

	user, err := rt.users.Register(r.Context(), request.Login, request.Password)
	switch {
	case err == nil:
		rt.authenticate(w, user.ID)
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrUserInvalidCredentialsFormat):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, domain.ErrUserAlreadyExists):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Router) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if rt.users == nil || rt.tokens == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	request, ok := readCredentials(w, r)
	if !ok {
		return
	}

	user, err := rt.users.Login(r.Context(), request.Login, request.Password)
	switch {
	case err == nil:
		rt.authenticate(w, user.ID)
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, domain.ErrUserInvalidCredentialsFormat):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, domain.ErrInvalidCredentials):
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Router) handleUploadOrder(w http.ResponseWriter, r *http.Request, userID int64) {
	if rt.orders == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = rt.orders.UploadOrder(r.Context(), userID, string(body))
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

func (rt *Router) handleListOrders(w http.ResponseWriter, r *http.Request, userID int64) {
	if rt.orders == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orders, err := rt.orders.ListOrders(r.Context(), userID)
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

type balanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type withdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type credentialsRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	var request credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return credentialsRequest{}, false
	}
	return request, true
}

func (rt *Router) authenticate(w http.ResponseWriter, userID int64) {
	token := rt.tokens.Issue(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Authorization", "Bearer "+token)
}

func (rt *Router) authorize(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if rt.tokens == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false
	}

	token := tokenFromRequest(r)
	userID, err := rt.tokens.Verify(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return 0, false
	}

	ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
	*r = *r.WithContext(ctx)
	return userID, true
}

func tokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(authCookieName); err == nil {
		return cookie.Value
	}

	const bearerPrefix = "Bearer "
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, bearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	}

	return ""
}

type userIDContextKey struct{}

func pointsPtrToFloat64(points *domain.Points) *float64 {
	if points == nil {
		return nil
	}
	value := points.Float64()
	return &value
}
