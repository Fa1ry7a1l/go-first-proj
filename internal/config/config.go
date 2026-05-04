// Package config reads runtime settings for the Gophermart service.
package config

import (
	"flag"
	"os"
)

const (
	defaultRunAddress = "localhost:8080"

	envRunAddress           = "RUN_ADDRESS"
	envDatabaseURI          = "DATABASE_URI"
	envAccrualSystemAddress = "ACCRUAL_SYSTEM_ADDRESS"
)

// Config contains runtime settings required to start the service.
type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
}

// Parse reads configuration from command-line flags and environment variables.
// Environment variables have priority over flags.
func Parse() Config {
	return ParseFromArgs(os.Args[1:], os.Getenv)
}

// ParseFromArgs reads configuration from the provided arguments and getenv
// function. It is intended for tests and for callers that need isolated flag
// parsing.
func ParseFromArgs(args []string, getenv func(string) string) Config {
	cfg := Config{
		RunAddress: defaultRunAddress,
	}

	flags := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	flags.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "HTTP server run address")
	flags.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "PostgreSQL database URI")
	flags.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address")

	_ = flags.Parse(args)

	if value := getenv(envRunAddress); value != "" {
		cfg.RunAddress = value
	}
	if value := getenv(envDatabaseURI); value != "" {
		cfg.DatabaseURI = value
	}
	if value := getenv(envAccrualSystemAddress); value != "" {
		cfg.AccrualSystemAddress = value
	}

	return cfg
}
