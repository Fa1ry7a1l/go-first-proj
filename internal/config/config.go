// Package config читает настройки запуска сервиса Gophermart.
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
	envAuthSecret           = "AUTH_SECRET"
)

// Config содержит настройки, необходимые для запуска сервиса.
type Config struct {
	// RunAddress содержит адрес и порт HTTP-сервера.
	RunAddress string

	// DatabaseURI содержит строку подключения к PostgreSQL.
	DatabaseURI string

	// AccrualSystemAddress содержит базовый адрес внешней системы начислений.
	AccrualSystemAddress string

	// AuthSecret содержит секрет для подписи токенов авторизации.
	AuthSecret string
}

// Parse читает конфигурацию из аргументов командной строки и переменных окружения.
// Переменные окружения имеют приоритет над флагами.
func Parse() Config {
	return ParseFromArgs(os.Args[1:], os.Getenv)
}

// ParseFromArgs читает конфигурацию из переданных аргументов и функции getenv.
// Функция нужна для тестов и случаев, где требуется изолированный разбор флагов.
func ParseFromArgs(args []string, getenv func(string) string) Config {
	cfg := Config{
		RunAddress: defaultRunAddress,
	}

	flags := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	flags.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "HTTP server run address")
	flags.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "PostgreSQL database URI")
	flags.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address")
	flags.StringVar(&cfg.AuthSecret, "s", cfg.AuthSecret, "auth token signing secret")

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
	if value := getenv(envAuthSecret); value != "" {
		cfg.AuthSecret = value
	}

	return cfg
}
