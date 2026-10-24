// Command golangcourse is the index and runner for this course.
//
//	go run .            list every lesson
//	go run . 14         run lesson 14
//	go run . channels   run the lesson matching "channels"
//	go run . all        run every lesson in order
//
// Each lesson is also just a normal program, so this works too:
//
//	go run ./lessons/14_channels
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type lesson struct {
	Dir   string // directory name, e.g. "14_channels"
	Title string
	Blurb string
}

var lessons = []lesson{
	{"01_variables", "Variables, constants & types", "declarations, zero values, conversion, iota enums"},
	{"02_control_flow", "Control flow", "if with initialiser, switch, type switch"},
	{"03_loops", "Loops", "every form of for, range over slices/maps/strings/ints/funcs"},
	{"04_functions", "Functions", "multiple returns, variadic, closures, defer"},
	{"05_arrays_slices", "Arrays & slices", "len vs cap, append growth, aliasing gotchas, the slices package"},
	{"06_maps", "Maps", "comma-ok, nil maps, sets, counting, stable iteration"},
	{"07_strings_runes", "Strings, bytes & runes", "UTF-8, strings.Builder, strconv"},
	{"08_pointers", "Pointers", "value semantics, receivers, escape analysis"},
	{"09_structs_methods", "Structs & methods", "constructors, embedding, tags, value vs pointer receivers"},
	{"10_interfaces", "Interfaces", "implicit satisfaction, dependency injection, type assertions, the nil trap"},
	{"11_errors", "Errors", "sentinels, custom types, %w wrapping, errors.Is/As, panic & recover"},
	{"12_generics", "Generics", "type parameters, constraints, generic containers"},
	{"13_goroutines", "Goroutines", "WaitGroup, data races, leaks, panic safety"},
	{"14_channels", "Channels", "buffered vs unbuffered, close, direction, pipelines"},
	{"15_select", "Select", "timeouts, non-blocking ops, tickers, done channels"},
	{"16_sync", "sync & atomic", "Mutex, RWMutex, Once, atomics, Pool, deadlock avoidance"},
	{"17_context", "Context", "cancellation, deadlines, propagation, request values"},
	{"18_concurrency_patterns", "Concurrency patterns", "worker pools, semaphores, fan-in/out, retry, single-flight"},
	{"19_packages_modules", "Packages & modules", "visibility, import paths, go.mod, project layout rules"},
	{"20_json", "JSON", "tags, omitempty, custom marshalling, streaming, RawMessage"},
	{"21_time", "Time", "durations, the reference layout, timers, monotonic clock"},
	{"22_files_io", "Files & io", "Reader/Writer composition, bufio, filepath, atomic writes"},
	{"23_sorting", "Sorting & searching", "slices.SortFunc, multi-key sorts, binary search, maps iterators"},
	{"24_testing", "Testing", "table tests, fakes, benchmarks, fuzzing, examples"},
	{"25_http_server", "HTTP server", "routing, middleware, JSON APIs, timeouts"},
	{"26_http_client", "HTTP client", "typed clients, retries, context timeouts, streaming"},
	{"27_reflection_tags", "Reflection & tags", "building a validator and a SQL generator from tags"},
	{"28_logging_config", "Logging & config", "log/slog structured logging, env configuration"},
	{"29_graceful_shutdown", "Graceful shutdown", "signals, draining, ordered cleanup"},
	{"30_project_layout", "Project layout", "how to structure a real application"},
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printIndex()
		return
	}

	if args[0] == "all" {
		for _, l := range lessons {
			fmt.Printf("\n\033[1;36m═══ %s: %s ═══\033[0m\n", l.Dir, l.Title)
			if err := runLesson(l); err != nil {
				fmt.Fprintf(os.Stderr, "lesson %s failed: %v\n", l.Dir, err)
			}
		}
		return
	}

	l, err := find(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "\nRun `go run .` to see the list of lessons.")
		os.Exit(1)
	}
	fmt.Printf("\033[1;36m═══ %s: %s ═══\033[0m\n", l.Dir, l.Title)
	if err := runLesson(*l); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printIndex() {
	fmt.Print(`
  GO COURSE - from the basics to shipping a service
  ─────────────────────────────────────────────────
  go run .          this index        go run . 14      run lesson 14
  go run . maps     run by name       go run . all     run everything

`)
	sections := []struct {
		name string
		from int
		to   int
	}{
		{"FOUNDATIONS", 1, 12},
		{"CONCURRENCY", 13, 18},
		{"STANDARD LIBRARY & TOOLING", 19, 24},
		{"BUILDING SERVICES", 25, 30},
	}
	for _, s := range sections {
		fmt.Printf("  \033[1m%s\033[0m\n", s.name)
		for i := s.from - 1; i < s.to && i < len(lessons); i++ {
			l := lessons[i]
			fmt.Printf("   %s  %-28s %s\n", l.Dir[:2], l.Title, l.Blurb)
		}
		fmt.Println()
	}
	fmt.Print(`  CAPSTONE PROJECT
   projects/taskapi   a complete, tested, layered HTTP service
                      go test ./projects/...
                      go run  ./projects/taskapi/cmd/api

  Read README.md for the suggested order and the exercises.

`)
}

// find resolves "14", "14_channels" or "channels" to a lesson.
func find(q string) (*lesson, error) {
	q = strings.ToLower(strings.TrimSpace(q))

	// Exact directory, or a numeric prefix like "7" / "07".
	if len(q) == 1 {
		q = "0" + q
	}
	for i := range lessons {
		if lessons[i].Dir == q || strings.HasPrefix(lessons[i].Dir, q+"_") {
			return &lessons[i], nil
		}
	}

	// Substring match on the directory, title or blurb.
	var matches []*lesson
	for i := range lessons {
		l := &lessons[i]
		haystack := strings.ToLower(l.Dir + " " + l.Title + " " + l.Blurb)
		if strings.Contains(haystack, q) {
			matches = append(matches, l)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no lesson matches %q", q)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.Dir)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("%q matches several lessons: %s", q, strings.Join(names, ", "))
	}
}

func runLesson(l lesson) error {
	// Shell out to `go run`, so each lesson stays an independent main package.
	cmd := exec.Command("go", "run", "./"+filepath.ToSlash(filepath.Join("lessons", l.Dir)))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
