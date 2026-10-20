// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	EmailDelay      time.Duration
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://hotel:hotel@localhost:5432/hotel?sslmode=disable"),
		EmailDelay:      time.Duration(getEnvInt("EMAIL_DELAY_SECONDS", 2)) * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v >= 0 {
		return v
	}
	return fallback
}
