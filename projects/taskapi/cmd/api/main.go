// Command api is the task service binary.
//
// main() stays deliberately thin: everything it does is wire dependencies
// together and hand control to run(). That makes the startup path readable
// and the whole program testable.
//
// Run: go run ./projects/taskapi/cmd/api
// Then, in another terminal:
//
//	curl localhost:8080/healthz
//	curl -X POST localhost:8080/v1/tasks -H 'Content-Type: application/json' \
//	     -d '{"title":"learn Go","priority":"high"}'
//	curl localhost:8080/v1/tasks
//	curl -X POST localhost:8080/v1/tasks/1/complete
//	curl localhost:8080/v1/tasks?done=true
//
// Stop it with Ctrl-C and watch it drain cleanly.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golangcourse/projects/taskapi/internal/config"
	"golangcourse/projects/taskapi/internal/httpapi"
	"golangcourse/projects/taskapi/internal/service"
	"golangcourse/projects/taskapi/internal/store"
)

// version is injected at build time:
//
//	go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" ./projects/taskapi/cmd/api
var version = "dev"

func main() {
	// os.Exit skips deferred functions, so it happens here and nowhere else.
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Configuration: load and validate before anything else exists.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// 2. Observability: JSON in production, human-readable while developing.
	logger := newLogger(cfg)
	logger.Info("starting",
		slog.String("version", version),
		slog.String("env", cfg.Env),
		slog.String("addr", cfg.Addr))

	// 3. A context cancelled by SIGINT/SIGTERM, threaded through everything.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Resources. Swapping in Postgres means changing only this line,
	//    because service depends on an interface, not on a database.
	tasks := store.NewMemory()
	// db, err := store.OpenPostgres(ctx, cfg.DatabaseURL)
	// if err != nil { return fmt.Errorf("database: %w", err) }
	// defer db.Close()

	// 5. Wire the layers: store -> service -> transport. By hand, explicitly.
	taskService := service.New(tasks, logger)
	srv := httpapi.NewServer(taskService, logger, cfg)

	// 6. Serve until a signal arrives, then drain (see lesson 29).
	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     cfg.LogLevel,
		AddSource: cfg.LogLevel == slog.LevelDebug,
	}
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
