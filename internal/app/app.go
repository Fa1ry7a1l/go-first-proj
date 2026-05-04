// Package app wires service dependencies and controls the application lifecycle.
package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/config"
	"github.com/Fa1ry7a1l/go-first-proj/internal/httpapi"
)

const shutdownTimeout = 5 * time.Second

// App is the runnable Gophermart application.
type App struct {
	cfg    config.Config
	server *http.Server
}

// New creates an application instance with all dependencies wired.
func New(cfg config.Config) *App {
	return &App{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.RunAddress,
			Handler:           httpapi.NewRouter(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run starts the HTTP server and blocks until the provided context is canceled
// or the server returns an unexpected error.
func (a *App) Run(ctx context.Context) error {
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
