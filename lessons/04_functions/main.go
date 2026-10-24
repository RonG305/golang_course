// Lesson 04: functions, multiple returns, defer, and closures.
//
// Run: go run ./lessons/04_functions
package main

import (
	"errors"
	"fmt"
	"strings"
)

// Multiple return values. Returning (result, error) is THE Go convention.
func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("cannot divide by zero")
	}
	return a / b, nil
}

// Shared type for consecutive params: (a, b float64) not (a float64, b float64).
func add(a, b int) int { return a + b }

// Named return values. Useful with defer, but don't overuse them - a bare
// `return` hides what is actually being returned.
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return // returns the current x, y
}

// Variadic: nums is a []int inside the function.
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// Functions are values, so they can be parameters and return types.
func apply(nums []int, f func(int) int) []int {
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		out = append(out, f(n))
	}
	return out
}

// A closure captures variables from its enclosing scope by reference.
func counter() func() int {
	count := 0 // lives as long as the returned function does
	return func() int {
		count++
		return count
	}
}

// defer runs when the function returns - LIFO order. With a named return it
// can even change the returned value, which is how error wrapping helpers work.
func process(name string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("process %q: %w", name, err)
		}
	}()
	if name == "" {
		return errors.New("empty name")
	}
	return nil
}

func main() {
	// The two-value call plus an immediate error check: you will write this
	// thousands of times. Never ignore err.
	result, err := divide(10, 2)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("10/2 =", result)

	if _, err := divide(1, 0); err != nil {
		fmt.Println("expected error:", err)
	}

	fmt.Println("add:", add(3, 4))

	x, y := split(17)
	fmt.Println("split:", x, y)

	fmt.Println("sum:", sum(1, 2, 3, 4))

	// Spread an existing slice into a variadic call with ...
	nums := []int{5, 6, 7}
	fmt.Println("spread:", sum(nums...))

	// Anonymous function passed inline.
	fmt.Println("squared:", apply(nums, func(n int) int { return n * n }))

	next := counter()
	fmt.Println("counter:", next(), next(), next())

	// Two closures from the same factory have independent state.
	a, b := counter(), counter()
	a()
	fmt.Println("independent:", a(), b())

	// defer is for cleanup that must happen on every exit path.
	demoDefer()

	fmt.Println("wrapped:", process(""))
}

func demoDefer() {
	var sb strings.Builder
	defer fmt.Println("deferred:", "runs last")
	for i := range 3 {
		defer fmt.Println("deferred:", i) // prints 2, 1, 0 - LIFO
	}
	sb.WriteString("body runs first")
	fmt.Println(sb.String())
}
