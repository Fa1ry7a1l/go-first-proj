package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
)

func TestRegisterAuthenticatesUser(t *testing.T) {
	storage := newHTTPFakeStorage()
	router := NewRouter(
		service.NewUserService(storage),
		service.NewOrderService(storage),
		service.NewBalanceService(storage),
		auth.NewTokenManager("secret"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"user","password":"secret"}`))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if len(response.Result().Cookies()) == 0 {
		t.Fatal("auth cookie was not set")
	}
	if response.Header().Get("Authorization") == "" {
		t.Fatal("Authorization header was not set")
	}
}

func TestRegisterRejectsDuplicateLogin(t *testing.T) {
	storage := newHTTPFakeStorage()
	router := NewRouter(
		service.NewUserService(storage),
		service.NewOrderService(storage),
		service.NewBalanceService(storage),
		auth.NewTokenManager("secret"),
	)

	first := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"user","password":"secret"}`))
	router.ServeHTTP(httptest.NewRecorder(), first)

	second := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"user","password":"secret"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, second)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	storage := newHTTPFakeStorage()
	userService := service.NewUserService(storage)
	router := NewRouter(userService, service.NewOrderService(storage), service.NewBalanceService(storage), auth.NewTokenManager("secret"))

	if _, err := userService.Register(context.Background(), "user", "secret"); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"user","password":"wrong"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestOrdersRequireAuthorization(t *testing.T) {
	storage := newHTTPFakeStorage()
	router := NewRouter(
		service.NewUserService(storage),
		service.NewOrderService(storage),
		service.NewBalanceService(storage),
		auth.NewTokenManager("secret"),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestUploadOrderUsesAuthenticatedUser(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if storage.orders["12345678903"].UserID != 12 {
		t.Fatalf("order user id = %d, want 12", storage.orders["12345678903"].UserID)
	}
}

func TestUploadOrderReadsGzipBody(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(gzipBytes(t, "12345678903")))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	request.Header.Set("Content-Encoding", "gzip")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
}

func TestUploadOrderRejectsBrokenGzipBody(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("not gzip"))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	request.Header.Set("Content-Encoding", "gzip")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestGetBalance(t *testing.T) {
	storage := newHTTPFakeStorage()
	points := domain.Points(12550)
	storage.orders["processed"] = domain.Order{
		UserID:  12,
		Number:  "12345678903",
		Status:  domain.OrderStatusProcessed,
		Accrual: &points,
	}
	storage.withdrawals = append(storage.withdrawals, domain.Withdrawal{
		UserID: 12,
		Sum:    2550,
	})
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body balanceResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	if body.Current != 100 || body.Withdrawn != 25.5 {
		t.Fatalf("balance = %+v, want current 100 and withdrawn 25.5", body)
	}
}

func TestGetBalanceWritesGzipResponse(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", response.Header().Get("Content-Encoding"))
	}

	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	if !strings.Contains(string(body), `"current":0`) {
		t.Fatalf("body = %s, want current field", body)
	}
}

func TestWithdraw(t *testing.T) {
	storage := newHTTPFakeStorage()
	points := domain.Points(10000)
	storage.orders["processed"] = domain.Order{
		UserID:  12,
		Number:  "12345678903",
		Status:  domain.OrderStatusProcessed,
		Accrual: &points,
	}
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"12345678903","sum":25.5}`))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if len(storage.withdrawals) != 1 {
		t.Fatalf("withdrawals = %d, want 1", len(storage.withdrawals))
	}
	if storage.withdrawals[0].Sum != 2550 {
		t.Fatalf("withdrawal sum = %d, want 2550", storage.withdrawals[0].Sum)
	}
}

func TestWithdrawRejectsInsufficientFunds(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"12345678903","sum":25.5}`))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
}

func TestWithdrawRejectsInvalidSum(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"12345678903","sum":0}`))
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestWithdrawalsReturnsNoContentWhenEmpty(t *testing.T) {
	storage := newHTTPFakeStorage()
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestWithdrawalsReturnsRFC3339Time(t *testing.T) {
	storage := newHTTPFakeStorage()
	storage.withdrawals = append(storage.withdrawals, domain.Withdrawal{
		UserID:      12,
		OrderNumber: "12345678903",
		Sum:         2550,
		ProcessedAt: time.Date(2026, 5, 5, 10, 11, 12, 0, time.UTC),
	})
	tokenManager := auth.NewTokenManager("secret")
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), service.NewBalanceService(storage), tokenManager)

	request := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	request.Header.Set("Authorization", "Bearer "+tokenManager.Issue(12))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body []withdrawalResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode withdrawals: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("withdrawals = %d, want 1", len(body))
	}
	if body[0].ProcessedAt != "2026-05-05T10:11:12Z" {
		t.Fatalf("processed_at = %q, want RFC3339", body[0].ProcessedAt)
	}
}

func gzipBytes(t *testing.T, value string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(value)); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buffer.Bytes()
}

type httpFakeStorage struct {
	nextUserID  int64
	users       map[string]domain.User
	orders      map[string]domain.Order
	withdrawals []domain.Withdrawal
}

func newHTTPFakeStorage() *httpFakeStorage {
	return &httpFakeStorage{
		nextUserID: 1,
		users:      make(map[string]domain.User),
		orders:     make(map[string]domain.Order),
	}
}

func (s *httpFakeStorage) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := s.users[user.Login]; ok {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	user.ID = s.nextUserID
	s.nextUserID++
	s.users[user.Login] = user
	return user, nil
}

func (s *httpFakeStorage) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	user, ok := s.users[login]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *httpFakeStorage) CreateOrder(_ context.Context, order domain.Order) error {
	if _, ok := s.orders[order.Number]; ok {
		return domain.ErrOrderUploadedByAnotherUser
	}
	s.orders[order.Number] = order
	return nil
}

func (s *httpFakeStorage) GetOrderByNumber(_ context.Context, number string) (domain.Order, error) {
	order, ok := s.orders[number]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return order, nil
}

func (s *httpFakeStorage) ListUserOrders(_ context.Context, userID int64) ([]domain.Order, error) {
	var orders []domain.Order
	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (s *httpFakeStorage) ListPendingOrders(_ context.Context, _ int) ([]domain.Order, error) {
	return nil, nil
}

func (s *httpFakeStorage) UpdateOrderAccrual(_ context.Context, _ string, _ domain.OrderStatus, _ *domain.Points) error {
	return nil
}

func (s *httpFakeStorage) GetBalance(_ context.Context, userID int64) (domain.Balance, error) {
	var accrued domain.Points
	for _, order := range s.orders {
		if order.UserID == userID && order.Status == domain.OrderStatusProcessed && order.Accrual != nil {
			accrued += *order.Accrual
		}
	}

	var withdrawn domain.Points
	for _, withdrawal := range s.withdrawals {
		if withdrawal.UserID == userID {
			withdrawn += withdrawal.Sum
		}
	}

	return domain.Balance{
		Current:   accrued - withdrawn,
		Withdrawn: withdrawn,
	}, nil
}

func (s *httpFakeStorage) CreateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error {
	balance, err := s.GetBalance(ctx, withdrawal.UserID)
	if err != nil {
		return err
	}
	if balance.Current < withdrawal.Sum {
		return domain.ErrInsufficientFunds
	}
	withdrawal.ID = int64(len(s.withdrawals) + 1)
	withdrawal.ProcessedAt = time.Now()
	s.withdrawals = append(s.withdrawals, withdrawal)
	return nil
}

func (s *httpFakeStorage) ListWithdrawals(_ context.Context, userID int64) ([]domain.Withdrawal, error) {
	var withdrawals []domain.Withdrawal
	for _, withdrawal := range s.withdrawals {
		if withdrawal.UserID == userID {
			withdrawals = append(withdrawals, withdrawal)
		}
	}
	return withdrawals, nil
}
