// Lesson 28: structured logging with log/slog, plus configuration.
//
// Structured logs are key-value pairs, not sentences. That is what makes them
// searchable in Loki/Datadog/CloudWatch. slog is in the standard library as
// of Go 1.21 - you no longer need zap or logrus for most services.
//
// Run: go run ./lessons/28_logging_config
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// --- 1. A text handler, good for local development ---
	dev := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	dev.Info("starting up", "version", "1.0.0", "port", 8080)
	dev.Debug("this shows because the level is Debug")

	// --- 2. A JSON handler, which is what you run in production ---
	fmt.Println("\n-- JSON output --")
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // true adds file:line - useful, slightly costly
		// ReplaceAttr lets you rename, redact or reformat any attribute.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "password" || a.Key == "token" {
				return slog.String(a.Key, "[REDACTED]")
			}
			if a.Key == slog.TimeKey {
				// Fixed timestamp here only so this lesson's output is stable.
				return slog.String("ts", "2026-01-01T00:00:00Z")
			}
			return a
		},
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	logger.Info("user logged in", "user_id", 42, "ip", "10.0.0.1", "token", "abc123")
	logger.Warn("slow query", "duration_ms", 1500, "table", "orders")
	logger.Error("payment failed", "err", errors.New("card declined"), "order_id", "A-1")
	logger.Debug("not printed: level is Info")

	// --- 3. Typed attributes avoid allocations and typos ---
	logger.Info("typed attrs",
		slog.String("service", "billing"),
		slog.Int("retries", 3),
		slog.Duration("took", 250*time.Millisecond),
		slog.Bool("cached", true),
		slog.Time("at", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
	)

	// --- 4. With(): a child logger that carries context on every line.
	//     This is the feature that makes structured logging worth it. ---
	reqLogger := logger.With(
		slog.String("request_id", "req-abc-123"),
		slog.String("route", "POST /orders"),
	)
	reqLogger.Info("handling request")
	reqLogger.Info("validated payload", "items", 3)
	reqLogger.Info("request complete", "status", 201, "duration_ms", 42)

	// --- 5. Groups nest related fields ---
	logger.Info("request finished",
		slog.Group("http",
			slog.String("method", "GET"),
			slog.Int("status", 200),
		),
		slog.Group("db", slog.Int("queries", 4), slog.Int("ms", 12)),
	)

	// --- 6. LogAttrs is the lowest-allocation form, for hot paths ---
	logger.LogAttrs(context.Background(), slog.LevelInfo, "hot path",
		slog.Int("count", 1))

	// --- 7. A custom type can control how it logs. Implement LogValuer to
	//     keep secrets out of your logs by construction. ---
	logger.Info("created user", "user", User{ID: 7, Email: "a@b.com", Password: "hunter2"})

	// --- 8. Setting the default logger makes the package-level calls structured ---
	slog.SetDefault(logger)
	slog.Info("via the default logger")

	// --- 9. Dynamic level: flip verbosity at run time without a redeploy ---
	fmt.Println("\n-- dynamic level --")
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)
	dyn := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: levelVar}))
	dyn.Debug("hidden at Info")
	levelVar.Set(slog.LevelDebug) // e.g. triggered by SIGHUP or an admin endpoint
	dyn.Debug("visible now that the level changed")

	// --- 10. Configuration from the environment ---
	fmt.Println("\n-- config --")
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", cfg)
	fmt.Println("DSN:", cfg.DSN())

	// Show validation rejecting bad input.
	os.Setenv("PORT", "not-a-port")
	if _, err := LoadConfig(); err != nil {
		fmt.Println("rejected bad PORT:", err)
	}
	os.Unsetenv("PORT")

	fmt.Println(`
--- Logging rules that keep you sane ---
  Log the ERROR where you HANDLE it, not where you create it. Wrap on the way up.
  One log line per event, with IDs you can grep (request_id, user_id, order_id).
  Never log secrets, tokens, passwords or full card numbers.
  No logging inside tight loops - log a summary instead.
  Pass a *slog.Logger into your structs; don't reach for a package global.
  Levels: Debug for development, Info for lifecycle, Warn for recoverable
  trouble, Error for things a human must look at. Reserve panics for bugs.`)
}

// --- LogValuer: control your type's log representation ---

type User struct {
	ID       int
	Email    string
	Password string
}

func (u User) LogValue() slog.Value {
	// Password simply cannot leak: it is not part of the representation.
	return slog.GroupValue(
		slog.Int("id", u.ID),
		slog.String("email", redactEmail(u.Email)),
	)
}

func redactEmail(e string) string {
	name, domain, found := strings.Cut(e, "@")
	if !found || name == "" {
		return "[invalid]"
	}
	return name[:1] + "***@" + domain
}

// --- Config: parse once at startup, validate, then pass it down ---

type Config struct {
	Env         string
	Port        int
	DBHost      string
	DBPort      int
	DBName      string
	LogLevel    slog.Level
	ReadTimeout time.Duration
	Debug       bool
}

func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%d/%s?sslmode=disable", c.DBHost, c.DBPort, c.DBName)
}

// LoadConfig reads the environment with defaults, then validates. Failing
// fast at startup beats discovering a typo at 3am under load.
func LoadConfig() (Config, error) {
	c := Config{
		Env:         env("APP_ENV", "development"),
		DBHost:      env("DB_HOST", "localhost"),
		DBName:      env("DB_NAME", "app"),
		ReadTimeout: envDuration("READ_TIMEOUT", 10*time.Second),
		Debug:       envBool("DEBUG", false),
	}

	var err error
	if c.Port, err = envInt("PORT", 8080); err != nil {
		return Config{}, err
	}
	if c.DBPort, err = envInt("DB_PORT", 5432); err != nil {
		return Config{}, err
	}
	c.LogLevel = parseLevel(env("LOG_LEVEL", "info"))

	// Validate: the whole point of loading config in one place.
	if c.Port < 1 || c.Port > 65535 {
		return Config{}, fmt.Errorf("PORT out of range: %d", c.Port)
	}
	if c.Env != "development" && c.Env != "staging" && c.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development|staging|production, got %q", c.Env)
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return n, nil
}

func envBool(key string, def bool) bool {
	if b, err := strconv.ParseBool(env(key, "")); err == nil {
		return b
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(env(key, "")); err == nil {
		return d
	}
	return def
}

func parseLevel(s string) slog.Level {
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
