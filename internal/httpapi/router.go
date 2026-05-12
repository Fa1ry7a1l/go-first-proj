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

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
)

const (
	authCookieName = "gophermart_auth"
)

// Router обрабатывает HTTP-запросы к API Gophermart.
type Router struct {
	users    UserService
	orders   OrderService
	balances BalanceService
	tokens   TokenService
}

// UserService описывает пользовательские операции, нужные HTTP-слою.
type UserService interface {
	// Register регистрирует нового пользователя.
	Register(ctx context.Context, login string, password string) (domain.User, error)

	// Login проверяет логин и пароль пользователя.
	Login(ctx context.Context, login string, password string) (domain.User, error)
}

// OrderService описывает операции с заказами, нужные HTTP-слою.
type OrderService interface {
	// UploadOrder загружает номер заказа пользователя.
	UploadOrder(ctx context.Context, userID int64, number string) error

	// ListOrders возвращает заказы пользователя.
	ListOrders(ctx context.Context, userID int64) ([]domain.Order, error)
}

// BalanceService описывает операции с балансом, нужные HTTP-слою.
type BalanceService interface {
	// GetBalance возвращает текущий баланс пользователя.
	GetBalance(ctx context.Context, userID int64) (domain.Balance, error)

	// Withdraw регистрирует списание баллов.
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum domain.Points) error

	// ListWithdrawals возвращает историю списаний пользователя.
	ListWithdrawals(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
}

// TokenService описывает операции с токенами авторизации, нужные HTTP-слою.
type TokenService interface {
	// Issue создает токен для пользователя.
	Issue(userID int64) string

	// Verify проверяет токен и возвращает идентификатор пользователя.
	Verify(token string) (int64, error)
}

// NewRouter создает дерево HTTP-обработчиков сервиса.
func NewRouter(
	users UserService,
	orders OrderService,
	balances BalanceService,
	tokens TokenService,
) http.Handler {
	router := &Router{
		users:    users,
		orders:   orders,
		balances: balances,
		tokens:   tokens,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("POST /api/user/register", router.handleRegister)
	mux.HandleFunc("POST /api/user/login", router.handleLogin)
	mux.HandleFunc("POST /api/user/orders", router.handleUploadOrder)
	mux.HandleFunc("GET /api/user/orders", router.handleListOrders)
	mux.HandleFunc("GET /api/user/balance", router.handleBalance)
	mux.HandleFunc("POST /api/user/balance/withdraw", router.handleWithdraw)
	mux.HandleFunc("GET /api/user/withdrawals", router.handleWithdrawals)
	return gzipMiddleware(mux)
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (rt *Router) handleBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
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
		w.WriteHeader(http.StatusUnprocessableEntity)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (rt *Router) handleWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
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

func (rt *Router) handleUploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}
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

func (rt *Router) handleListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := rt.authorize(w, r)
	if !ok {
		return
	}
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
