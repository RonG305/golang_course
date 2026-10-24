// Lesson 17: context - cancellation, deadlines and request scoping.
//
// Every blocking or long-running function in a serious Go codebase takes
// ctx as its FIRST parameter. context is how a cancelled request tears down
// all the work it started, all the way down the call stack.
//
// Run: go run ./lessons/17_context
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

func main() {
	// --- context.Background(): the root, used in main/init/tests ---
	ctx := context.Background()

	// --- WithCancel: you decide when to stop ---
	ctx1, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel() // every goroutine watching ctx1 is released
	}()
	fmt.Println("cancel:", poll(ctx1, "A"))
	fmt.Println("ctx.Err():", ctx1.Err())

	// ALWAYS defer cancel(), even when the context will expire on its own -
	// it releases the timer and the parent's child list. Not doing so leaks.

	// --- WithTimeout: the one you will use most ---
	ctx2, cancel2 := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel2()
	fmt.Println("timeout:", poll(ctx2, "B"))
	fmt.Println("is DeadlineExceeded:", errors.Is(ctx2.Err(), context.DeadlineExceeded))

	// --- WithDeadline: same thing, absolute time ---
	ctx3, cancel3 := context.WithDeadline(ctx, time.Now().Add(20*time.Millisecond))
	defer cancel3()
	if dl, ok := ctx3.Deadline(); ok {
		fmt.Printf("deadline in %.0fms\n", time.Until(dl).Seconds()*1000)
	}

	// --- A task that finishes before cancellation ---
	ctx4, cancel4 := context.WithTimeout(ctx, time.Second)
	defer cancel4()
	fmt.Println("fast work:", poll(ctx4, "C"))

	// --- Cancellation PROPAGATES to every derived context ---
	parent, cancelParent := context.WithCancel(ctx)
	child, cancelChild := context.WithTimeout(parent, time.Hour)
	defer cancelChild()
	go func() { time.Sleep(10 * time.Millisecond); cancelParent() }()
	<-child.Done() // the child dies with its parent despite the 1h timeout
	fmt.Println("child cancelled because parent was:", child.Err())

	// --- Respecting ctx inside a loop: check between units of work ---
	ctx5, cancel5 := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel5()
	done, err := processItems(ctx5, 100)
	fmt.Printf("processed %d items, stopped because: %v\n", done, err)

	// --- WithValue: request-scoped metadata only (trace/request IDs, auth).
	//     Never use it for optional parameters - pass those explicitly. ---
	ctx6 := context.WithValue(ctx, requestIDKey, "req-abc-123")
	handle(ctx6)

	// --- Fanning a context out to a worker pool ---
	ctx7, cancel7 := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel7()
	fmt.Println("pool completed:", runPool(ctx7, 4))

	// --- AfterFunc (Go 1.21+): run a callback when a context ends ---
	ctx8, cancel8 := context.WithCancel(ctx)
	stop := context.AfterFunc(ctx8, func() { fmt.Println("cleanup ran on cancel") })
	defer stop()
	cancel8()
	time.Sleep(10 * time.Millisecond)

	// --- WithoutCancel (Go 1.21+): keep values but detach from cancellation,
	//     for work that must outlive the request (audit logs, metrics) ---
	detached := context.WithoutCancel(ctx6)
	fmt.Println("detached keeps values:", detached.Value(requestIDKey), "err:", detached.Err())
}

// poll is the shape of every cancellable operation: select on your work
// AND on ctx.Done().
func poll(ctx context.Context, name string) string {
	work := time.After(20 * time.Millisecond)
	select {
	case <-work:
		return name + ": completed"
	case <-ctx.Done():
		return name + ": aborted (" + ctx.Err().Error() + ")"
	}
}

func processItems(ctx context.Context, n int) (int, error) {
	for i := range n {
		// Cheap, non-blocking cancellation check.
		select {
		case <-ctx.Done():
			return i, ctx.Err() // return the context error, wrapped if you add detail
		default:
		}
		time.Sleep(time.Millisecond) // pretend this is real work
	}
	return n, nil
}

// --- Typed, unexported keys prevent collisions between packages ---
type ctxKey string

const requestIDKey ctxKey = "requestID"

func requestID(ctx context.Context) string {
	// Always use the comma-ok form: the value may be missing or another type.
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return "unknown"
}

func handle(ctx context.Context) {
	fmt.Println("handling request:", requestID(ctx))
	nested(ctx) // values flow down the whole call stack
}

func nested(ctx context.Context) {
	fmt.Println("  nested still sees:", requestID(ctx))
}

func runPool(ctx context.Context, workers int) int {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		completed int
	)
	for i := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			select {
			case <-time.After(time.Duration(10+id*20) * time.Millisecond):
				mu.Lock()
				completed++
				mu.Unlock()
			case <-ctx.Done():
				// Slower workers bail out instead of finishing pointless work.
			}
		}(i)
	}
	wg.Wait()
	return completed
}
