package config

import "testing"

func TestParseFromArgsUsesDefaults(t *testing.T) {
	cfg := ParseFromArgs(nil, emptyEnv)

	if cfg.RunAddress != defaultRunAddress {
		t.Fatalf("RunAddress = %q, want %q", cfg.RunAddress, defaultRunAddress)
	}
	if cfg.DatabaseURI != "" {
		t.Fatalf("DatabaseURI = %q, want empty", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "" {
		t.Fatalf("AccrualSystemAddress = %q, want empty", cfg.AccrualSystemAddress)
	}
	if cfg.AuthSecret != "" {
		t.Fatalf("AuthSecret = %q, want empty", cfg.AuthSecret)
	}
}

func TestParseFromArgsReadsFlags(t *testing.T) {
	cfg := ParseFromArgs([]string{
		"-a", "127.0.0.1:9090",
		"-d", "postgres://user:pass@localhost/db",
		"-r", "http://localhost:8081",
		"-s", "flag-secret",
	}, emptyEnv)

	if cfg.RunAddress != "127.0.0.1:9090" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://user:pass@localhost/db" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://localhost:8081" {
		t.Fatalf("AccrualSystemAddress = %q", cfg.AccrualSystemAddress)
	}
	if cfg.AuthSecret != "flag-secret" {
		t.Fatalf("AuthSecret = %q", cfg.AuthSecret)
	}
}

func TestParseFromArgsEnvOverridesFlags(t *testing.T) {
	env := map[string]string{
		envRunAddress:           "localhost:7070",
		envDatabaseURI:          "postgres://env/db",
		envAccrualSystemAddress: "http://accrual",
		envAuthSecret:           "env-secret",
	}

	cfg := ParseFromArgs([]string{
		"-a", "127.0.0.1:9090",
		"-d", "postgres://flag/db",
		"-r", "http://flag-accrual",
		"-s", "flag-secret",
	}, func(key string) string {
		return env[key]
	})

	if cfg.RunAddress != env[envRunAddress] {
		t.Fatalf("RunAddress = %q, want %q", cfg.RunAddress, env[envRunAddress])
	}
	if cfg.DatabaseURI != env[envDatabaseURI] {
		t.Fatalf("DatabaseURI = %q, want %q", cfg.DatabaseURI, env[envDatabaseURI])
	}
	if cfg.AccrualSystemAddress != env[envAccrualSystemAddress] {
		t.Fatalf("AccrualSystemAddress = %q, want %q", cfg.AccrualSystemAddress, env[envAccrualSystemAddress])
	}
	if cfg.AuthSecret != env[envAuthSecret] {
		t.Fatalf("AuthSecret = %q, want %q", cfg.AuthSecret, env[envAuthSecret])
	}
}

func emptyEnv(string) string {
	return ""
}
