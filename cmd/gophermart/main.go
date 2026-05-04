package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Fa1ry7a1l/go-first-proj/internal/app"
	"github.com/Fa1ry7a1l/go-first-proj/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Parse()
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
