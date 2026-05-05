package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
)

func TestRegisterAuthenticatesUser(t *testing.T) {
	storage := newHTTPFakeStorage()
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), auth.NewTokenManager("secret"))

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
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), auth.NewTokenManager("secret"))

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
	router := NewRouter(userService, service.NewOrderService(storage), auth.NewTokenManager("secret"))

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
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), auth.NewTokenManager("secret"))

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
	router := NewRouter(service.NewUserService(storage), service.NewOrderService(storage), tokenManager)

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

type httpFakeStorage struct {
	nextUserID int64
	users      map[string]domain.User
	orders     map[string]domain.Order
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
