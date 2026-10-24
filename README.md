# Go, from the basics to shipping a service

A hands-on course in one Go module. Every lesson is a **runnable program** with
commented source — read the code, run it, change it, break it.

```bash
go run .              # the index of all 30 lessons
go run . 14           # run lesson 14
go run . channels     # run by name
go run . all          # run everything in order

go run ./lessons/14_channels     # or run a lesson directly
go test ./...                    # run every test in the course
```

Requires Go 1.22+ (1.23+ for the iterator parts). Check with `go version`.

---

## Layout

```
golang_course/
├── main.go              the course index / runner
├── lessons/             30 self-contained lessons, one package each
└── projects/taskapi/    the capstone: a complete, tested HTTP service
```

---

## The path

Work through it in order. Each part assumes the one before.

### Part 1 — Foundations (lessons 01–12)

| # | Lesson | What you take away |
|---|--------|--------------------|
| 01 | Variables & types | zero values, explicit conversion, `iota` enums, `%v`/`%+v`/`%T` |
| 02 | Control flow | `if` with an initialiser, `switch` without `break`, type switches |
| 03 | Loops | one keyword covers every loop; `range` over slices, maps, strings, ints, funcs |
| 04 | Functions | `(result, error)`, variadics, closures, `defer` |
| 05 | Arrays & slices | `len` vs `cap`, how `append` reallocates, **the aliasing traps** |
| 06 | Maps | the comma-ok idiom, nil maps, sets, deterministic iteration |
| 07 | Strings & runes | bytes vs code points, `strings.Builder`, `strconv` |
| 08 | Pointers | everything is passed by value; value vs pointer receivers |
| 09 | Structs & methods | constructors, **embedding instead of inheritance**, struct tags |
| 10 | Interfaces | implicit satisfaction, dependency injection, the typed-nil trap |
| 11 | Errors | sentinels, custom types, `%w`, `errors.Is`/`As`, `panic`/`recover` |
| 12 | Generics | type parameters, constraints, when *not* to use them |

The three that matter most for writing real Go: **05** (slices), **10**
(interfaces) and **11** (errors). If something later confuses you, it is usually
one of those three.

### Part 2 — Concurrency (lessons 13–18)

| # | Lesson | What you take away |
|---|--------|--------------------|
| 13 | Goroutines | `WaitGroup`, data races, goroutine leaks, panic safety |
| 14 | Channels | buffered vs unbuffered, `close`, direction types, pipelines |
| 15 | Select | timeouts, non-blocking sends, tickers, done channels |
| 16 | sync & atomic | `Mutex`, `RWMutex`, `Once`, atomics, deadlock avoidance |
| 17 | Context | cancellation and deadlines through the whole call stack |
| 18 | Patterns | worker pools, semaphores, fan-in/out, retry, single-flight |

Run these with `-race`. The race detector is the single most valuable tool in
the Go toolchain:

```bash
go run -race ./lessons/13_goroutines    # deliberately reports a race
go test -race ./...
```

Two rules that prevent most concurrency bugs:

1. **Bound your concurrency.** Never start one goroutine per item of unbounded
   input — use a worker pool or a semaphore (lesson 18).
2. **Every goroutine needs a way to exit.** A `close`, a context, a timeout.
   Otherwise it leaks (lesson 13).

### Part 3 — Standard library & tooling (lessons 19–24)

| # | Lesson | What you take away |
|---|--------|--------------------|
| 19 | Packages & modules | visibility, import paths, `internal/`, the `go` commands |
| 20 | JSON | tags, `omitempty`, custom marshalling, streaming decoders |
| 21 | Time | durations, the reference layout, timers, the monotonic clock |
| 22 | Files & io | `io.Reader`/`Writer` composition, `bufio`, atomic writes |
| 23 | Sorting | `slices.SortFunc`, multi-key sorts, binary search |
| 24 | Testing | table tests, hand-written fakes, benchmarks, fuzzing, examples |

Lesson 24 has a real test suite in `lessons/24_testing/cart/`:

```bash
go test -v ./lessons/24_testing/...
go test -bench=. -benchmem ./lessons/24_testing/cart
go test -fuzz=FuzzAdd -fuzztime=10s ./lessons/24_testing/cart
```

### Part 4 — Building services (lessons 25–30)

| # | Lesson | What you take away |
|---|--------|--------------------|
| 25 | HTTP server | Go 1.22+ routing, middleware, JSON APIs, server timeouts |
| 26 | HTTP client | typed API clients, retry with backoff, context timeouts |
| 27 | Reflection & tags | build a validator and a SQL generator from struct tags |
| 28 | Logging & config | `log/slog` structured logging, validated env config |
| 29 | Graceful shutdown | signals, draining in-flight requests, ordered cleanup |
| 30 | Project layout | how to structure an application that keeps growing |

### Part 5 — The capstone

`projects/taskapi/` is a complete task-management HTTP service that uses
everything above in one codebase: layered architecture, interfaces defined by
their consumers, an injected clock, structured logging, graceful shutdown, and
tests at every layer.

```bash
go test -race ./projects/...              # all layers
go run ./projects/taskapi/cmd/api         # then curl it (see its README)
```

Read it in dependency order: `domain` → `store` → `service` → `httpapi` →
`cmd/api`. Its [README](projects/taskapi/README.md) has the full API and a
guided tour.

---

## Exercises

Do these in order. They are deliberately the things that break when you have
only read about them.

**Foundations**

1. Write `func WordCount(s string) map[string]int`. Make it count runes, not
   bytes, and lowercase the keys.
2. Trigger the slice aliasing bug from lesson 05 on purpose, then fix it with a
   three-index slice.
3. Define a `Stack[T]` with `Push`, `Pop`, `Peek` and `Len`, where `Pop` on an
   empty stack returns `(zero, false)` instead of panicking.
4. Write a `Shape` interface with three implementations, then a function that
   returns the largest by area — without a type switch.
5. Build an error chain three layers deep, with a sentinel at the bottom and a
   custom type in the middle. Prove `errors.Is` and `errors.As` both find them.

**Concurrency**

6. Fetch 20 URLs concurrently, at most 4 at a time, with a 2-second timeout on
   the whole batch. Return partial results when the deadline hits.
7. Write a worker pool that drains cleanly on `ctx` cancellation with no
   goroutine left behind. Prove it with `runtime.NumGoroutine()`.
8. Build a concurrency-safe LRU cache (`Mutex` + map + list). Test it with
   `-race` and 100 goroutines.
9. Take the race in `lessons/13_goroutines/main.go:unsafeCounter` and fix it
   three ways: mutex, atomic, and channel. Benchmark all three.

**Services**

10. Add `PATCH /v1/tasks/{id}` to the capstone. Use pointer fields so that
    "not provided" and "set to empty" are distinguishable.
11. Add a `Postgres` store behind the existing `service.TaskStore` interface.
    Every existing test must still pass against `Memory`.
12. Add token auth as middleware, with the user ID in the request context.
13. Add a rate limiter: 10 requests/second per IP, returning 429.
14. Write a `Dockerfile` (multi-stage, `FROM scratch`) and get the binary under
    15 MB with `-ldflags="-s -w"`.

---

## The tooling to use daily

```bash
gofmt -w .                    # formatting is not a matter of taste in Go
go vet ./...                  # built-in correctness checks
go test -race -cover ./...    # always -race in CI
go build ./...
go doc net/http.Server        # offline docs for anything
go mod tidy                   # sync go.mod with the imports

# worth installing
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

`make help` lists the shortcuts in the Makefile.

---

## Habits that make Go code good

- **Handle every error where it happens.** Wrap with `fmt.Errorf("doing x: %w", err)`
  on the way up; handle it once, at the top.
- **Accept interfaces, return structs.** And define the interface in the package
  that *uses* it, listing only the methods it needs.
- **Keep interfaces small.** One or two methods. `io.Writer` is the ideal.
- **`ctx context.Context` is the first parameter** of anything that blocks.
- **No global mutable state.** Pass dependencies in explicitly; wire them in
  `main`.
- **Zero values should work.** A `sync.Mutex`, a `bytes.Buffer` and a nil slice
  are all usable as-is. Design your types the same way.
- **Prefer clear over clever.** Go has no ternary and no method overloading on
  purpose. Write the boring version.
- **Read the standard library.** `net/http`, `io`, `errors` and `sync` are the
  best Go style guide there is.

## Where to go after this

- [Effective Go](https://go.dev/doc/effective_go) — still the best single document
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — the review checklist Go teams actually use
- [The Go Memory Model](https://go.dev/ref/mem) — read it before writing lock-free code
- [Go by Example](https://gobyexample.com) — quick reference for any construct
- `go doc <package>` — faster than a search engine, and always matches your version
