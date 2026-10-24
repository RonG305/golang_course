// Lesson 30: how to lay out a real Go application.
//
// Run: go run ./lessons/30_project_layout
//
// The working example of everything described here is in ../../projects/taskapi.
// Read it after this: it is a small but complete, layered, tested service.
package main

import "fmt"

func main() {
	fmt.Print(`
================================================================
 STRUCTURING A GO APPLICATION
================================================================

A layout that scales. Nothing here is enforced by the toolchain
except internal/ - the rest is convention that the community has
converged on, and it works.

  myservice/
    go.mod
    go.sum
    Makefile                 build / test / lint entry points
    README.md
    .golangci.yml            linter config
    cmd/
      api/main.go            ONE binary per directory, tiny main()
      worker/main.go         a second binary sharing the same internal code
      migrate/main.go
    internal/                private to this module - the COMPILER enforces it
      config/config.go       load + validate env/flags, once, at startup
      domain/                entities and business rules, NO framework imports
        task.go
        errors.go
      store/                 persistence adapters
        postgres.go
        memory.go            an in-memory implementation for tests
      service/              use cases: orchestrates domain + store
        task_service.go
      httpapi/              transport: routing, decode, encode, middleware
        server.go
        handlers.go
        middleware.go
      worker/               background jobs, queue consumers
    pkg/                     optional: code you genuinely want others to import
    migrations/              .sql files
    api/                     OpenAPI spec, protobuf definitions
    deployments/             Dockerfile, k8s manifests, compose files
    scripts/
    testdata/                fixtures (the go tool ignores this directory)

----------------------------------------------------------------
 THE DEPENDENCY RULE
----------------------------------------------------------------

Dependencies point INWARD. Nothing in the inner layers imports
anything from the outer ones:

  cmd  ->  httpapi  ->  service  ->  domain
                \-->    store    ->  domain

  * domain knows nothing about HTTP, SQL or JSON.
  * service depends on domain plus INTERFACES it declares itself.
  * store and httpapi are interchangeable adapters.
  * cmd is the only place that knows the concrete types, and it
    wires them together.

Go rewards this because import cycles are a COMPILE ERROR. If you
feel the urge to import service from domain, the design is wrong.

----------------------------------------------------------------
 DEFINE INTERFACES WHERE THEY ARE CONSUMED
----------------------------------------------------------------

Do NOT put a big Repository interface next to the implementation.
The consumer declares the narrow interface it actually needs:

  // in internal/service - the CONSUMER owns this
  type TaskStore interface {
      Create(ctx context.Context, t domain.Task) (domain.Task, error)
      ByID(ctx context.Context, id string) (domain.Task, error)
  }

Now service has no dependency on postgres at all, and a test can
pass a 20-line fake. This is the single most valuable habit in Go.

----------------------------------------------------------------
 main() SHOULD BE BORING
----------------------------------------------------------------

  func main() {
      if err := run(); err != nil {        // no os.Exit deep in the code
          fmt.Fprintln(os.Stderr, err)     // so defers always run
          os.Exit(1)
      }
  }

  func run() error {
      cfg, err := config.Load()            // 1. config, fail fast
      logger := slog.New(...)              // 2. observability
      db, err := store.OpenPostgres(cfg)   // 3. resources
      defer db.Close()
      svc := service.New(db, logger)       // 4. wire dependencies
      srv := httpapi.NewServer(svc, logger)
      return srv.Run(ctx)                  // 5. run until a signal (lesson 29)
  }

Wire by hand. Go projects rarely need a DI framework - explicit
construction in run() is clearer and it compiles.

----------------------------------------------------------------
 THINGS TO AVOID
----------------------------------------------------------------

  * Global mutable state and init(). Pass dependencies explicitly.
  * A util/, common/ or helpers/ package: it becomes a dumping
    ground that everything imports. Name packages for what they
    PROVIDE (store, httpapi, retry, slugs).
  * Stuttering names: user.UserService reads as user.Service.
  * One giant package. Also: a package per file. Group by what
    changes together.
  * Returning database models from HTTP handlers. Map to a DTO so
    a schema change cannot break your API contract.
  * Business logic in handlers. Handlers decode, call a service,
    and encode. That is all.

----------------------------------------------------------------
 THE TOOLING TO SET UP ON DAY ONE
----------------------------------------------------------------

  gofmt -w .                      formatting is not a debate in Go
  go vet ./...                    built-in correctness checks
  go test -race -cover ./...      always -race in CI
  golangci-lint run               aggregates ~50 linters
  go build -ldflags="-s -w"       smaller binaries
  go tool pprof                   CPU and memory profiling
  govulncheck ./...               known vulnerabilities in your deps

----------------------------------------------------------------
 WHERE TO GO NEXT
----------------------------------------------------------------

  1. Read  projects/taskapi/  in this repo: it implements all of
     the above in a working, tested service.
  2. Then add, as practice:
       - a Postgres store behind the same interface (database/sql + pgx)
       - JWT or session auth as middleware
       - request validation and consistent error responses
       - metrics (prometheus/client_golang) and tracing (OpenTelemetry)
       - a Dockerfile (multi-stage, FROM scratch or distroless)
       - a CI pipeline running vet, lint, test -race
  3. Read the standard library source. It is the best Go style
     guide in existence: net/http, io, errors, sync are all worth
     an afternoon each.

`)
}
