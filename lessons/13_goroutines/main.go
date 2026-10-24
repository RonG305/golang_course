// Lesson 13: goroutines - concurrency primitives that cost ~2KB each.
//
// `go f()` starts f in a new goroutine and returns immediately. The tricky part
// is never the starting - it is knowing when the work is DONE and keeping
// shared data safe.
//
// Run: go run ./lessons/13_goroutines
//
// Then run it again with the race detector:
//
//	go run -race ./lessons/13_goroutines
//
// It WILL report a data race, on purpose - unsafeCounter() below is broken.
// Get used to running -race on your tests; it finds bugs nothing else will.
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	fmt.Println("CPUs:", runtime.NumCPU(), "goroutines at start:", runtime.NumGoroutine())

	// --- A goroutine with no synchronisation may never run: main exits first ---
	go fmt.Println("this might not print")
	time.Sleep(10 * time.Millisecond) // sleeping is NOT synchronisation - see below

	// --- WaitGroup: the standard way to wait for N goroutines ---
	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1) // increment BEFORE starting the goroutine
		go func() {
			defer wg.Done() // defer guarantees it runs even on panic
			fmt.Printf("worker %d\n", i)
		}()
	}
	wg.Wait() // blocks until the counter hits zero
	fmt.Println("all workers finished")

	// --- The loop-variable gotcha (fixed in Go 1.22) ---
	// Before 1.22 every goroutine shared ONE `i`, so they all printed 5.
	// Go 1.22+ gives each iteration a fresh variable. If you ever read older
	// code, you will see `go func(i int){...}(i)` written to work around it.
	var wg2 sync.WaitGroup
	results := make([]int, 3)
	for i := range 3 {
		wg2.Add(1)
		go func(idx int) { // explicit parameter: works on every Go version
			defer wg2.Done()
			results[idx] = idx * idx // distinct indices, so no race
		}(i)
	}
	wg2.Wait()
	fmt.Println("squares:", results)

	// --- Collecting results: channels avoid shared-memory races entirely ---
	fmt.Println("sum via channel:", concurrentSum([]int{1, 2, 3, 4, 5, 6, 7, 8}))

	// --- Why unsynchronised writes are broken ---
	fmt.Println("unsafe counter:", unsafeCounter()) // usually < 1000
	fmt.Println("mutex counter:", mutexCounter())   // always 1000

	// --- Goroutine leaks: the most common production bug ---
	// A goroutine blocked forever on a channel is never collected.
	leaky()
	time.Sleep(10 * time.Millisecond)
	fmt.Println("goroutines now (note the leak):", runtime.NumGoroutine())

	// --- Fire-and-forget still needs panic protection: an unrecovered panic
	//     in ANY goroutine kills the whole process ---
	var wg3 sync.WaitGroup
	wg3.Add(1)
	go safeGo(&wg3, func() { panic("boom") })
	wg3.Wait()
	fmt.Println("survived the panic")
}

func concurrentSum(nums []int) int {
	mid := len(nums) / 2
	ch := make(chan int, 2) // buffered so neither half blocks
	for _, part := range [][]int{nums[:mid], nums[mid:]} {
		go func(p []int) {
			s := 0
			for _, n := range p {
				s += n
			}
			ch <- s
		}(part)
	}
	return <-ch + <-ch // receive both halves
}

func unsafeCounter() int {
	count := 0
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count++ // DATA RACE: read-modify-write is not atomic
		}()
	}
	wg.Wait()
	return count
}

func mutexCounter() int {
	count := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			count++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return count
}

// leaky starts a goroutine that blocks forever. Nobody ever sends, so it
// cannot exit. In a server this accumulates until you run out of memory.
// The fix is always: give the goroutine a way out (close, context, timeout).
func leaky() {
	ch := make(chan int) // unbuffered, no sender
	go func() {
		<-ch // blocked forever
		fmt.Println("never reached")
	}()
}

func safeGo(wg *sync.WaitGroup, f func()) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered in goroutine:", r)
		}
	}()
	f()
}
