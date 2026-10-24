# taskapi — the capstone service

A small but complete HTTP service that applies everything from the lessons:
layered architecture, interfaces owned by their consumers, structured logging,
context propagation, graceful shutdown, and tests at every layer.

```bash
go test -race ./projects/...              # all tests
go run ./projects/taskapi/cmd/api         # start it on 127.0.0.1:8080
```

## Try it

```bash
curl localhost:8080/healthz

curl -X POST localhost:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"learn Go","priority":"high","due_at":"2027-01-01T00:00:00Z"}'

curl localhost:8080/v1/tasks
curl 'localhost:8080/v1/tasks?priority=high&limit=10'
curl localhost:8080/v1/tasks/1
curl -X POST localhost:8080/v1/tasks/1/complete
curl -X DELETE localhost:8080/v1/tasks/1
curl localhost:8080/v1/reports/overdue

# validation errors come back structured, per field
curl -X POST localhost:8080/v1/tasks \
  -H 'Content-Type: application/json' -d '{"title":"","priority":"urgent"}'
```

Then stop it with Ctrl-C and watch it drain in the logs.

## Endpoints

| Method | Path | Returns |
|--------|------|---------|
| GET | `/healthz` | liveness — is the process up |
| GET | `/readyz` | readiness — are dependencies reachable |
| POST | `/v1/tasks` | 201 + `Location`, or 422 with per-field errors |
| GET | `/v1/tasks` | paged list; `?done=&priority=&limit=&offset=` |
| GET | `/v1/tasks/{id}` | 200 or 404 |
| POST | `/v1/tasks/{id}/complete` | 200, or 409 if already done |
| DELETE | `/v1/tasks/{id}` | 204 or 404 |
| GET | `/v1/reports/overdue` | a cross-record report |

Configuration, all optional:

```bash
ADDR=127.0.0.1:9000 APP_ENV=production LOG_LEVEL=debug \
  go run ./projects/taskapi/cmd/api
```

## The layers

```
cmd/api  ──▶  httpapi  ──▶  service  ──▶  domain
                   └──────▶  store  ──────▶  domain
```

Dependencies point inward only. Import cycles are a compile error in Go, so the
language itself keeps this honest.

| Package | Responsibility | Knows about |
|---------|----------------|-------------|
| `domain` | entities, validation, business rules | nothing else |
| `store` | persistence (`Memory` here) | `domain` |
| `service` | use cases, orchestration | `domain` + interfaces it declares |
| `httpapi` | routing, decode/encode, error mapping | `service`, `domain` |
| `config` | load + validate configuration once | nothing else |
| `cmd/api` | wire everything, run until a signal | all of them |

## The ideas worth copying

**The consumer declares the interface.** `service.TaskStore` lives in
`internal/service/task_service.go`, not next to the store. The store package has
never heard of it. Swapping in Postgres means writing a new type with the same
methods and changing one line in `cmd/api/main.go`.

**Validation lives in the domain.** `domain.NewTask` is the only way to build a
`Task`, and it returns `FieldErrors` listing *everything* wrong at once. An
invalid `Task` cannot exist in the system, and handlers do no validation at all.

**Errors are mapped in exactly one place.** `Server.writeServiceError` is the
only function that turns a domain error into a status code
(`ErrNotFound` → 404, `ErrAlreadyDone` → 409, `FieldErrors` → 422, anything
unrecognised → 500 with the detail logged, never sent). The mapping cannot drift
between endpoints.

**The clock is a dependency.** `service.WithClock` lets `TestOverdueReport` jump
48 hours forward instantly. No `time.Sleep` in any test, so nothing is flaky.

**The transport has its own types.** `httpapi/dto.go` maps domain entities to
response structs, so renaming a domain field cannot silently break the API, and
internal state never leaks to clients.

**main() is boring.** `main` calls `run() error` and that is all, so deferred
cleanup always runs. `run` does five things in order: config, logger, resources,
wiring, serve.

## Tests

| File | Covers |
|------|--------|
| `domain/task_test.go` | pure rules — no I/O, microseconds to run |
| `store/memory_test.go` | the storage contract, including a `-race` concurrency test |
| `service/task_service_test.go` | use cases, with an injected clock and a failing-store fake |
| `httpapi/handlers_test.go` | the full stack via `httptest`, no network needed |

```bash
go test -race ./projects/...
go test -v ./projects/taskapi/internal/httpapi
go test -bench=. -benchmem ./projects/taskapi/...
go test -cover ./projects/...
```

The handler tests drive `Server.Handler()` through the complete middleware chain
with `httptest.NewRecorder`, which is why they can assert on request IDs and
panic recovery without binding a port.

## What to add next

Each of these is a realistic next step, and the architecture is already shaped
for it:

1. **A Postgres store** — implement `service.TaskStore` with `database/sql` or
   `pgx`. Every existing service and handler test must pass unchanged.
2. **`PATCH /v1/tasks/{id}`** — use pointer fields in the request DTO so
   "absent" and "set to empty" are different.
3. **Auth middleware** — validate a token, put the user ID in the context, scope
   queries to it.
4. **Rate limiting** — per-IP, returning 429 with a `Retry-After` header.
5. **Metrics and tracing** — `prometheus/client_golang`, then OpenTelemetry
   spans around the service methods.
6. **A Dockerfile** — multi-stage, `FROM scratch`, `CGO_ENABLED=0`,
   `-ldflags="-s -w"`.
