// Lesson 18: the concurrency patterns you will actually ship.
//
// Run: go run ./lessons/18_concurrency_patterns
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Job struct {
	ID  int
	Arg int
}

type Result struct {
	JobID int
	Value int
	Err   error
}

func main() {
	ctx := context.Background()

	// 1. WORKER POOL - bound your concurrency. Never start one goroutine per
	//    item for unbounded input: you will exhaust memory, sockets or the DB.
	fmt.Println("--- worker pool ---")
	jobs := make([]Job, 9)
	for i := range jobs {
		jobs[i] = Job{ID: i + 1, Arg: i + 1}
	}
	for _, r := range workerPool(ctx, jobs, 3) {
		if r.Err != nil {
			fmt.Printf("job %d failed: %v\n", r.JobID, r.Err)
			continue
		}
		fmt.Printf("job %d -> %d\n", r.JobID, r.Value)
	}

	// 2. SEMAPHORE - limit concurrency without a pool, using a buffered channel
	fmt.Println("--- semaphore ---")
	fmt.Println("max 2 at a time, peak observed:", semaphoreDemo(2, 6))

	// 3. PIPELINE - stages connected by channels, each stage owns its output
	fmt.Println("--- pipeline ---")
	for v := range filterEven(squareStage(numberGen(ctx, 1, 2, 3, 4, 5, 6))) {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// 4. FAN-OUT / FAN-IN - parallelise one slow stage across N goroutines
	fmt.Println("--- fan-out/fan-in ---")
	src := numberGen(ctx, 1, 2, 3, 4, 5, 6, 7, 8)
	workers := make([]<-chan int, 3)
	for i := range workers {
		workers[i] = squareStage(src) // all three read from the SAME source
	}
	sum := 0
	for v := range fanIn(workers...) {
		sum += v
	}
	fmt.Println("sum of squares:", sum)

	// 5. ERRGROUP BY HAND - run tasks in parallel, stop on the first error
	fmt.Println("--- first error wins ---")
	err := runAll(ctx,
		func(context.Context) error { time.Sleep(10 * time.Millisecond); return nil },
		func(context.Context) error { return errors.New("task 2 broke") },
		func(ctx context.Context) error {
			select {
			case <-time.After(time.Second):
				return errors.New("should have been cancelled")
			case <-ctx.Done():
				return nil // cancelled cleanly
			}
		},
	)
	fmt.Println("group err:", err)

	// 6. TIMEOUT WRAPPER around any blocking call
	fmt.Println("--- timeout ---")
	v, err := withTimeout(20*time.Millisecond, func() (int, error) {
		time.Sleep(5 * time.Millisecond)
		return 42, nil
	})
	fmt.Println("in time:", v, err)
	_, err = withTimeout(10*time.Millisecond, func() (int, error) {
		time.Sleep(time.Second)
		return 0, nil
	})
	fmt.Println("too slow:", err)

	// 7. RETRY WITH BACKOFF - every network call needs this
	fmt.Println("--- retry ---")
	attempts := 0
	err = retry(ctx, 5, time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})
	fmt.Printf("succeeded after %d attempts (err=%v)\n", attempts, err)

	// 8. DEBOUNCE / RATE LIMIT with a ticker
	fmt.Println("--- rate limit ---")
	fmt.Println("5 calls at 1 per 5ms took:", rateLimited(5, 5*time.Millisecond).Round(time.Millisecond))

	// 9. SINGLE-FLIGHT - collapse duplicate concurrent work on the same key
	fmt.Println("--- single flight ---")
	sf := &singleFlight{calls: map[string]*call{}}
	var wg sync.WaitGroup
	hits := 0
	var mu sync.Mutex
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = sf.Do("key", func() (int, error) {
				mu.Lock()
				hits++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				return 1, nil
			})
		}()
	}
	wg.Wait()
	fmt.Printf("5 concurrent callers caused %d real call(s)\n", hits)
}

// --- 1. Worker pool ---
func workerPool(ctx context.Context, jobs []Job, workers int) []Result {
	jobCh := make(chan Job)
	resCh := make(chan Result)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh { // every worker pulls from one shared channel
				select {
				case <-ctx.Done():
					return
				default:
				}
				resCh <- process(j)
			}
		}()
	}

	// Feed the workers, then close so their range loops end.
	go func() {
		defer close(jobCh)
		for _, j := range jobs {
			select {
			case jobCh <- j:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close the results channel once every worker has exited.
	go func() {
		wg.Wait()
		close(resCh)
	}()

	var out []Result
	for r := range resCh {
		out = append(out, r)
	}
	return out
}

func process(j Job) Result {
	if j.Arg%4 == 0 {
		return Result{JobID: j.ID, Err: fmt.Errorf("arg %d not supported", j.Arg)}
	}
	return Result{JobID: j.ID, Value: j.Arg * j.Arg}
}

// --- 2. Semaphore ---
func semaphoreDemo(limit, tasks int) int {
	sem := make(chan struct{}, limit) // capacity IS the limit
	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		active, max int
	)
	for range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}        // acquire (blocks when full)
			defer func() { <-sem }() // release
			mu.Lock()
			active++
			if active > max {
				max = active
			}
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			active--
			mu.Unlock()
		}()
	}
	wg.Wait()
	return max
}

// --- 3/4. Pipeline stages ---
func numberGen(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done(): // never block forever on a cancelled pipeline
				return
			}
		}
	}()
	return out
}

func squareStage(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func filterEven(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			if n%2 == 0 {
				out <- n
			}
		}
	}()
	return out
}

func fanIn(chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, c := range chans {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}
	go func() { wg.Wait(); close(out) }()
	return out
}

// --- 5. Run all, first error wins (this is golang.org/x/sync/errgroup) ---
func runAll(ctx context.Context, fns ...func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		once    sync.Once
		firstEr error
	)
	for _, fn := range fns {
		wg.Add(1)
		go func(fn func(context.Context) error) {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				once.Do(func() {
					firstEr = err
					cancel() // tell the siblings to give up
				})
			}
		}(fn)
	}
	wg.Wait()
	return firstEr
}

// --- 6. Timeout wrapper ---
func withTimeout[T any](d time.Duration, fn func() (T, error)) (T, error) {
	type res struct {
		v   T
		err error
	}
	ch := make(chan res, 1) // buffered: the goroutine can exit even after a timeout
	go func() {
		v, err := fn()
		ch <- res{v, err}
	}()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-time.After(d):
		var zero T
		return zero, fmt.Errorf("timed out after %v", d)
	}
}

// --- 7. Retry with exponential backoff and jitter ---
func retry(ctx context.Context, maxAttempts int, base time.Duration, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		backoff := base * time.Duration(1<<(attempt-1))          // 1x, 2x, 4x, 8x...
		jitter := time.Duration(rand.Int63n(int64(backoff) + 1)) // avoid thundering herds
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, err)
}

// --- 8. Rate limiting ---
func rateLimited(calls int, every time.Duration) time.Duration {
	start := time.Now()
	limiter := time.Tick(every) // fine for a demo; use x/time/rate in production
	for range calls {
		<-limiter
	}
	return time.Since(start)
}

// --- 9. Single flight ---
type call struct {
	wg  sync.WaitGroup
	val int
	err error
}

type singleFlight struct {
	mu    sync.Mutex
	calls map[string]*call
}

func (sf *singleFlight) Do(key string, fn func() (int, error)) (int, error) {
	sf.mu.Lock()
	if c, ok := sf.calls[key]; ok {
		sf.mu.Unlock()
		c.wg.Wait() // someone else is already doing it: wait and share the result
		return c.val, c.err
	}
	c := &call{}
	c.wg.Add(1)
	sf.calls[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	sf.mu.Lock()
	delete(sf.calls, key)
	sf.mu.Unlock()
	return c.val, c.err
}
