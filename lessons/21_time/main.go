// Lesson 21: time - durations, formatting, timers.
//
// Run: go run ./lessons/21_time
package main

import (
	"fmt"
	"time"
)

func main() {
	// --- Instants ---
	now := time.Now()
	fmt.Println("year:", now.Year(), "month:", now.Month(), "weekday:", now.Weekday())
	fmt.Println("unix seconds:", now.Unix(), "| millis:", now.UnixMilli())

	// Build a specific instant. ALWAYS be explicit about the location.
	t := time.Date(2026, time.March, 15, 14, 30, 0, 0, time.UTC)
	fmt.Println("constructed:", t)

	// --- Duration is just an int64 count of nanoseconds, with units ---
	d := 90 * time.Minute
	fmt.Println("duration:", d, "| hours:", d.Hours(), "| minutes:", d.Minutes())
	fmt.Println("parsed:", must(time.ParseDuration("1h30m10s")))

	// GOTCHA: multiplying a Duration by a variable needs a conversion.
	n := 5
	fmt.Println("n seconds:", time.Duration(n)*time.Second)

	// --- Arithmetic ---
	fmt.Println("in 48h:", t.Add(48*time.Hour).Format(time.DateOnly))
	fmt.Println("calendar months:", t.AddDate(0, 2, 0).Format(time.DateOnly)) // handles month lengths
	later := t.Add(3 * time.Hour)
	fmt.Println("difference:", later.Sub(t))
	fmt.Println("before/after/equal:", t.Before(later), t.After(later), t.Equal(t))

	// --- Formatting uses a REFERENCE LAYOUT, not %Y-%m-%d.
	//     The magic reference date is: Mon Jan 2 15:04:05 MST 2006
	//     (i.e. 01/02 03:04:05PM '06 -0700) ---
	fmt.Println("custom:", t.Format("2006-01-02 15:04:05"))
	fmt.Println("readable:", t.Format("Mon, 02 Jan 2006 at 3:04 PM"))
	fmt.Println("RFC3339:", t.Format(time.RFC3339)) // use this for APIs and logs
	fmt.Println("constants:", t.Format(time.DateOnly), "|", t.Format(time.TimeOnly))

	// --- Parsing: the layout must match the input exactly ---
	p, err := time.Parse(time.RFC3339, "2026-03-15T14:30:00Z")
	fmt.Println("parsed:", p.Format(time.DateOnly), err)
	_, err = time.Parse(time.DateOnly, "15/03/2026")
	fmt.Println("mismatch error:", err != nil)

	// --- Time zones ---
	nairobi, err := time.LoadLocation("Africa/Nairobi")
	if err == nil {
		fmt.Println("in Nairobi:", t.In(nairobi).Format("15:04 MST"))
	}
	fmt.Println("UTC vs Local:", t.UTC().Format(time.TimeOnly), t.Local().Format(time.TimeOnly))
	// Store and transmit UTC; convert to a local zone only for display.

	// --- Truncate / Round: bucketing for metrics ---
	fmt.Println("truncated to hour:", t.Truncate(time.Hour).Format(time.TimeOnly))
	fmt.Println("rounded to hour:", t.Round(time.Hour).Format(time.TimeOnly))

	// --- Measuring elapsed time: Since uses the MONOTONIC clock, so it is
	//     immune to the wall clock jumping (NTP, DST) ---
	start := time.Now()
	work()
	fmt.Println("elapsed:", time.Since(start).Round(time.Millisecond))

	// A reusable timing helper via defer.
	func() {
		defer timed("block")()
		work()
	}()

	// --- Sleep, Timer, Ticker ---
	time.Sleep(5 * time.Millisecond)

	timer := time.NewTimer(10 * time.Millisecond)
	<-timer.C
	fmt.Println("timer fired")
	timer.Stop() // harmless once fired; essential if you abandon it early

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop() // a leaked ticker leaks a goroutine and a timer
	count := 0
	for range ticker.C {
		count++
		if count == 3 {
			break
		}
	}
	fmt.Println("ticks:", count)

	// time.AfterFunc: run something later without managing a goroutine.
	done := make(chan struct{})
	time.AfterFunc(5*time.Millisecond, func() { close(done) })
	<-done
	fmt.Println("AfterFunc ran")

	// --- Deadlines as values you pass around ---
	deadline := time.Now().Add(20 * time.Millisecond)
	fmt.Println("budget left:", time.Until(deadline).Round(time.Millisecond) > 0)

	// --- time.Time contains a wall clock, a monotonic reading and a location,
	//     so compare with Equal, not ==, and never use it as a map key
	//     without .UTC().Truncate(0) first ---
	a := time.Now()
	b := a.Round(0) // strips the monotonic reading
	fmt.Println("Equal:", a.Equal(b), "| == :", a == b)

	// --- Zero value means "no time set" ---
	var zero time.Time
	fmt.Println("zero is zero:", zero.IsZero())
}

func work() { time.Sleep(3 * time.Millisecond) }

// timed returns a function that prints the elapsed time: defer timed("x")()
func timed(label string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", label, time.Since(start).Round(time.Millisecond))
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
