// Package config loads and validates configuration exactly once, at startup.
// Everything downstream receives a plain, already-valid struct.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Addr            string
	LogLevel        slog.Level
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Load reads the environment, applies defaults, then validates. Returning an
// error (rather than exiting) keeps this package testable.
func Load() (Config, error) {
	c := Config{
		Env:             str("APP_ENV", "development"),
		Addr:            str("ADDR", "127.0.0.1:8080"),
		LogLevel:        level(str("LOG_LEVEL", "info")),
		ReadTimeout:     dur("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    dur("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     dur("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: dur("SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	switch c.Env {
	case "development", "staging", "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development|staging|production, got %q", c.Env)
	}
	if !strings.Contains(c.Addr, ":") {
		return Config{}, fmt.Errorf("ADDR must include a port, got %q", c.Addr)
	}
	if _, portStr, _ := strings.Cut(c.Addr, ":"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err != nil || port < 0 || port > 65535 {
			return Config{}, fmt.Errorf("ADDR has an invalid port: %q", portStr)
		}
	}
	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func str(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func dur(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(str(key, "")); err == nil {
		return d
	}
	return def
}

func level(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
