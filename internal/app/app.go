// Package app связывает зависимости сервиса и управляет жизненным циклом приложения.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/accrual"
	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/config"
	"github.com/Fa1ry7a1l/go-first-proj/internal/httpapi"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage/postgres"
)

const (
	shutdownTimeout       = 5 * time.Second
	developmentAuthSecret = "gophermart-development-auth-secret"
)

// App представляет запускаемое приложение Gophermart.
type App struct {
	cfg    config.Config
	server *http.Server
	worker *accrual.Worker
	close  func()
}

// New создает экземпляр приложения и подключает необходимые зависимости.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	var orderService *service.OrderService
	var userService *service.UserService
	var balanceService *service.BalanceService
	var accrualWorker *accrual.Worker
	closeStorage := func() {}

	if cfg.DatabaseURI != "" {
		storage, err := postgres.New(ctx, cfg.DatabaseURI)
		if err != nil {
			return nil, err
		}
		orderService = service.NewOrderService(storage)
		userService = service.NewUserService(storage)
		balanceService = service.NewBalanceService(storage)
		if cfg.AccrualSystemAddress != "" {
			accrualWorker = accrual.NewWorker(storage, accrual.NewClient(cfg.AccrualSystemAddress))
		}
		closeStorage = storage.Close
	}
	authSecret := cfg.AuthSecret
	if authSecret == "" {
		slog.Warn(
			"секрет подписи токенов не задан, используется dev-секрет",
			"source",
			"config",
		)
		authSecret = developmentAuthSecret
	}
	tokenManager := auth.NewTokenManager(authSecret)

	return &App{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.RunAddress,
			Handler:           httpapi.NewRouter(userService, orderService, balanceService, tokenManager),
			ReadHeaderTimeout: 5 * time.Second,
		},
		worker: accrualWorker,
		close:  closeStorage,
	}, nil
}

// Run запускает HTTP-сервер и блокируется до отмены контекста или неожиданной
// ошибки сервера.
func (a *App) Run(ctx context.Context) error {
	defer a.close()

	errCh := make(chan error, 1)

	if a.worker != nil {
		go a.worker.Run(ctx)
	}

	go func() {
		slog.Info("запуск gophermart", "address", a.cfg.RunAddress)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}
