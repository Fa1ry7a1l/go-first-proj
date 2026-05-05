// Package app связывает зависимости сервиса и управляет жизненным циклом приложения.
package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/auth"
	"github.com/Fa1ry7a1l/go-first-proj/internal/config"
	"github.com/Fa1ry7a1l/go-first-proj/internal/httpapi"
	"github.com/Fa1ry7a1l/go-first-proj/internal/service"
	"github.com/Fa1ry7a1l/go-first-proj/internal/storage/postgres"
)

const shutdownTimeout = 5 * time.Second

// App представляет запускаемое приложение Gophermart.
type App struct {
	cfg    config.Config
	server *http.Server
	close  func()
}

// New создает экземпляр приложения и подключает необходимые зависимости.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	var orderService *service.OrderService
	var userService *service.UserService
	var balanceService *service.BalanceService
	closeStorage := func() {}

	if cfg.DatabaseURI != "" {
		storage, err := postgres.New(ctx, cfg.DatabaseURI)
		if err != nil {
			return nil, err
		}
		orderService = service.NewOrderService(storage)
		userService = service.NewUserService(storage)
		balanceService = service.NewBalanceService(storage)
		closeStorage = storage.Close
	}
	tokenManager := auth.NewTokenManager("")

	return &App{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.RunAddress,
			Handler:           httpapi.NewRouter(userService, orderService, balanceService, tokenManager),
			ReadHeaderTimeout: 5 * time.Second,
		},
		close: closeStorage,
	}, nil
}

// Run запускает HTTP-сервер и блокируется до отмены контекста или неожиданной
// ошибки сервера.
func (a *App) Run(ctx context.Context) error {
	defer a.close()

	errCh := make(chan error, 1)

	go func() {
		log.Printf("starting gophermart on %s", a.cfg.RunAddress)
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
