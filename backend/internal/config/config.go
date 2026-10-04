// Package config loads runtime configuration from environment variables.
// Keep this the single place that reads os.Getenv in the whole app —
// everything else receives config values through structs, not env lookups.
package config

import (
	"fmt"
	"os"
)

// DefaultAllowedOrigin is the CORS origin assumed when ALLOWED_ORIGIN is unset:
// the Next.js dev server. Set ALLOWED_ORIGIN in production instead of editing code.
const DefaultAllowedOrigin = "http://localhost:3000"

type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisDB       int
	LLMAPIKey     string
	LLMModel      string
	Env           string // "development" | "production"
	AllowedOrigin string // origin permitted to call the API via CORS
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		LLMAPIKey:     getEnv("LLM_API_KEY", ""),
		LLMModel:      getEnv("LLM_MODEL", "gemini-3.8-flash"),
		Env:           getEnv("APP_ENV", "development"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", DefaultAllowedOrigin),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
