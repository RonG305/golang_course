// Lesson 16: the sync and sync/atomic packages.
//
// Channels are for handing work and data between goroutines. Mutexes are for
// protecting a piece of shared state in place. Use whichever makes the code
// clearer - for a cache or a counter, a mutex usually wins.
//
// Run: go run -race ./lessons/16_sync
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// --- The canonical pattern: embed the mutex next to the data it guards ---
type SafeCounter struct {
	mu     sync.Mutex // guards counts
	counts map[string]int
}

func NewSafeCounter() *SafeCounter {
	return &SafeCounter{counts: make(map[string]int)}
}

func (c *SafeCounter) Inc(key string) {
	c.mu.Lock()
	defer c.mu.Unlock() // defer keeps it unlocked on every path, panics included
	c.counts[key]++
}

func (c *SafeCounter) Get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[key]
}

// Never return the internal map: callers could read it without the lock.
func (c *SafeCounter) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.counts))
	for k, v := range c.counts {
		out[k] = v
	}
	return out
}

// --- RWMutex: many concurrent readers, one exclusive writer.
//
//	Worth it for read-heavy data like a config or a cache. ---
type Config struct {
	mu   sync.RWMutex
	data map[string]string
}

func (c *Config) Get(k string) string {
	c.mu.RLock() // other readers may hold RLock at the same time
	defer c.mu.RUnlock()
	return c.data[k]
}

func (c *Config) Set(k, v string) {
	c.mu.Lock() // excludes all readers and writers
	defer c.mu.Unlock()
	c.data[k] = v
}

// --- sync.Once: lazy, exactly-once initialisation (even under concurrency) ---
type db struct{ name string }

var (
	instance *db
	once     sync.Once
	initRuns int32
)

func getDB() *db {
	once.Do(func() {
		atomic.AddInt32(&initRuns, 1)
		time.Sleep(time.Millisecond) // pretend connecting is slow
		instance = &db{name: "main-db"}
	})
	return instance
}

func main() {
	// --- Mutex-protected map under load ---
	c := NewSafeCounter()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Inc("total")
			if i%2 == 0 {
				c.Inc("even")
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("counts:", c.Snapshot())

	// --- RWMutex ---
	cfg := &Config{data: map[string]string{"env": "dev"}}
	var wg2 sync.WaitGroup
	for range 10 {
		wg2.Add(1)
		go func() { defer wg2.Done(); _ = cfg.Get("env") }()
	}
	wg2.Add(1)
	go func() { defer wg2.Done(); cfg.Set("env", "prod") }()
	wg2.Wait()
	fmt.Println("env:", cfg.Get("env"))

	// --- sync.Once from 50 goroutines: the body runs exactly once ---
	var wg3 sync.WaitGroup
	for range 50 {
		wg3.Add(1)
		go func() { defer wg3.Done(); _ = getDB() }()
	}
	wg3.Wait()
	fmt.Printf("db=%s, init ran %d time(s)\n", instance.name, atomic.LoadInt32(&initRuns))

	// --- sync/atomic: lock-free counters. Faster than a mutex for one value. ---
	var ops atomic.Int64 // the typed API (Go 1.19+) is the one to prefer
	var wg4 sync.WaitGroup
	for range 1000 {
		wg4.Add(1)
		go func() { defer wg4.Done(); ops.Add(1) }()
	}
	wg4.Wait()
	fmt.Println("atomic ops:", ops.Load())

	// Compare-and-swap: the building block of lock-free algorithms.
	var flag atomic.Bool
	fmt.Println("CAS false->true:", flag.CompareAndSwap(false, true))
	fmt.Println("CAS again:", flag.CompareAndSwap(false, true)) // false, already true

	// atomic.Value / atomic.Pointer for swapping whole snapshots atomically.
	var current atomic.Pointer[Config]
	current.Store(cfg)
	fmt.Println("atomic pointer:", current.Load().Get("env"))

	// --- sync.Map: only for high-contention, mostly-read, disjoint-key cases.
	//     A plain map + Mutex is the right default. ---
	var sm sync.Map
	sm.Store("a", 1)
	sm.Store("b", 2)
	if v, ok := sm.Load("a"); ok {
		fmt.Println("sync.Map a:", v)
	}
	actual, loaded := sm.LoadOrStore("c", 3)
	fmt.Println("LoadOrStore:", actual, "already there:", loaded)
	n := 0
	sm.Range(func(k, v any) bool { n++; return true })
	fmt.Println("sync.Map entries:", n)

	// --- sync.Pool: reuse allocations in hot paths (buffers, encoders) ---
	pool := sync.Pool{New: func() any { return make([]byte, 0, 1024) }}
	b := pool.Get().([]byte)
	b = append(b, "reused buffer"...)
	fmt.Println("pool:", string(b))
	pool.Put(b[:0]) // reset length before putting it back

	// --- Deadlock avoidance ---
	// 1. Never call an unknown function while holding a lock.
	// 2. If you need two locks, always take them in the same global order.
	// 3. sync.Mutex is NOT reentrant: locking twice in one goroutine deadlocks.
	fmt.Println("ordered locking:", transfer())
}

type account struct {
	mu      sync.Mutex
	id      int
	balance int
}

// transfer locks by ascending id, so two concurrent transfers in opposite
// directions can never deadlock against each other.
func transfer() int {
	a, b := &account{id: 1, balance: 100}, &account{id: 2, balance: 50}
	move := func(from, to *account, amount int) {
		first, second := from, to
		if first.id > second.id {
			first, second = second, first
		}
		first.mu.Lock()
		defer first.mu.Unlock()
		second.mu.Lock()
		defer second.mu.Unlock()
		from.balance -= amount
		to.balance += amount
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); move(a, b, 10) }()
	go func() { defer wg.Done(); move(b, a, 5) }()
	wg.Wait()
	return a.balance + b.balance // conserved: always 150
}
