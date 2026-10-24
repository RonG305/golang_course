// Lesson 15: select - waiting on several channel operations at once.
//
// select blocks until one of its cases can proceed. If several are ready it
// picks one at random, which prevents starvation.
//
// Run: go run ./lessons/15_select
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// --- Whichever channel is ready first wins ---
	fast := after(10*time.Millisecond, "fast")
	slow := after(50*time.Millisecond, "slow")
	select {
	case v := <-fast:
		fmt.Println("winner:", v)
	case v := <-slow:
		fmt.Println("winner:", v)
	}

	// --- Timeout: the single most useful select pattern ---
	select {
	case v := <-after(100*time.Millisecond, "late"):
		fmt.Println("got:", v)
	case <-time.After(20 * time.Millisecond):
		fmt.Println("timed out waiting for work") // this one fires
	}

	// --- Non-blocking send/receive with default ---
	ch := make(chan int, 1)
	select {
	case v := <-ch:
		fmt.Println("received", v)
	default:
		fmt.Println("nothing to receive, moving on")
	}

	ch <- 1
	select {
	case ch <- 2:
		fmt.Println("sent")
	default:
		fmt.Println("buffer full, dropped the value") // load shedding
	}

	// --- Loop + select until a done channel closes: the worker skeleton ---
	work := make(chan int)
	quit := make(chan struct{})
	go worker(work, quit)
	for i := range 3 {
		work <- i
	}
	close(quit)
	time.Sleep(10 * time.Millisecond)

	// --- Ticker: do something periodically. ALWAYS Stop it. ---
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(50 * time.Millisecond)
	ticks := 0
loop:
	for {
		select {
		case <-ticker.C:
			ticks++
		case <-timeout:
			break loop // `break` alone would only exit the select
		}
	}
	fmt.Println("ticks:", ticks)

	// --- Draining a channel without blocking ---
	q := make(chan int, 3)
	q <- 1
	q <- 2
	fmt.Println("drained:", drain(q))

	// --- A nil channel blocks forever, so you can disable a case dynamically ---
	a := make(chan int, 1)
	var b chan int // nil
	a <- 1
	for range 1 {
		select {
		case v := <-a:
			fmt.Println("from a:", v)
		case v := <-b:
			fmt.Println("never happens:", v)
		}
	}

	// --- First result wins: cancel the losers so they don't leak ---
	fmt.Println("fastest replica:", fastestOf(3))
}

func after(d time.Duration, msg string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		time.Sleep(d)
		ch <- msg
	}()
	return ch
}

func worker(work <-chan int, quit <-chan struct{}) {
	for {
		select {
		case v := <-work:
			fmt.Println("worker handled:", v)
		case <-quit:
			fmt.Println("worker shutting down cleanly")
			return
		}
	}
}

func drain(ch <-chan int) []int {
	var out []int
	for {
		select {
		case v := <-ch:
			out = append(out, v)
		default:
			return out // empty right now
		}
	}
}

// fastestOf queries n replicas and returns whichever answers first. The
// channel is buffered for ALL of them so the slow ones can still send and
// exit instead of blocking forever - that is how you avoid a leak here.
func fastestOf(n int) string {
	results := make(chan string, n)
	for i := range n {
		go func(id int) {
			time.Sleep(time.Duration(rand.Intn(20)+1) * time.Millisecond)
			results <- fmt.Sprintf("replica-%d", id)
		}(i)
	}
	return <-results
}
