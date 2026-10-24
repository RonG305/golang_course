// Lesson 14: channels - typed pipes for communication between goroutines.
//
// "Don't communicate by sharing memory; share memory by communicating."
//
// Run: go run ./lessons/14_channels
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	// --- Unbuffered: send and receive RENDEZVOUS. The sender blocks until a
	//     receiver is ready, which makes it a synchronisation point. ---
	done := make(chan string)
	go func() {
		time.Sleep(10 * time.Millisecond)
		done <- "work finished" // blocks until main receives
	}()
	fmt.Println("received:", <-done)

	// --- Buffered: sends succeed until the buffer is full ---
	buf := make(chan int, 3)
	buf <- 1
	buf <- 2
	fmt.Println("len/cap:", len(buf), cap(buf))
	fmt.Println("got:", <-buf, <-buf)

	// --- Directional types document intent and are enforced by the compiler ---
	nums := make(chan int, 5)
	go produce(nums, 5) // produce only sends
	consume(nums)       // consume only receives

	// --- close + range: the idiomatic producer/consumer stream ---
	// The producer closes; the consumer's range ends automatically.
	stream := generate(1, 2, 3)
	for v := range stream {
		fmt.Print("stream:", v, " ")
	}
	fmt.Println()

	// --- Receiving from a closed channel: zero value, immediately, forever.
	//     The comma-ok form distinguishes "closed" from "zero was sent". ---
	c := make(chan int, 1)
	c <- 7
	close(c)
	v, ok := <-c
	fmt.Println("buffered value after close:", v, ok)
	v, ok = <-c
	fmt.Println("drained:", v, ok)

	// Rules you must respect, or you panic:
	//   - sending on a closed channel panics
	//   - closing a closed channel panics
	//   - only the SENDER closes, never the receiver
	//   - a nil channel blocks forever (occasionally useful in select)

	// --- Pipeline: each stage is a goroutine, channels are the joints ---
	for v := range square(generate(1, 2, 3, 4)) {
		fmt.Print("squared:", v, " ")
	}
	fmt.Println()

	// --- Signalling with a struct{} channel: zero bytes, pure notification ---
	ready := make(chan struct{})
	go func() {
		fmt.Println("worker: setting up")
		close(ready) // closing broadcasts to EVERY waiter at once
	}()
	<-ready
	fmt.Println("main: worker is ready")

	// --- Fan-in: many producers, one channel, closed once via WaitGroup ---
	fmt.Println("merged total:", sumAll(merge(generate(1, 2), generate(3, 4), generate(5))))

	// --- Deadlock demo (reasoned about, not executed) ---
	// ch := make(chan int)
	// ch <- 1  // fatal error: all goroutines are asleep - deadlock!
	// Unbuffered sends need a CONCURRENT receiver. There is no receiver here.
	fmt.Println("done")
}

// chan<- int: send-only. Attempting to receive is a compile error.
func produce(out chan<- int, n int) {
	for i := 1; i <= n; i++ {
		out <- i * 10
	}
	close(out)
}

// <-chan int: receive-only.
func consume(in <-chan int) {
	total := 0
	for v := range in {
		total += v
	}
	fmt.Println("consumed total:", total)
}

func generate(vals ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // ALWAYS close in the producer, with defer
		for _, v := range vals {
			out <- v
		}
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

func merge(chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, ch := range chans {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}
	// One closer goroutine: waits for every input to drain, then closes.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func sumAll(in <-chan int) int {
	total := 0
	for v := range in {
		total += v
	}
	return total
}
