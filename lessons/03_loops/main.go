// Lesson 03: for is Go's only loop keyword - it covers every case.
//
// Run: go run ./lessons/03_loops
package main

import "fmt"

func main() {
	// --- Classic three-clause loop ---
	for i := 0; i < 3; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// --- Condition only: this is Go's "while" ---
	count := 0
	for count < 3 {
		count++
	}
	fmt.Println("count:", count)

	// --- No condition: infinite, exit with break/return ---
	n := 0
	for {
		n++
		if n == 5 {
			break
		}
	}
	fmt.Println("n:", n)

	// --- range over a slice: index and value ---
	nums := []int{10, 20, 30}
	for i, v := range nums {
		fmt.Printf("nums[%d]=%d ", i, v)
	}
	fmt.Println()

	// Only need the value? Use _ for the index.
	sum := 0
	for _, v := range nums {
		sum += v
	}
	fmt.Println("sum:", sum)

	// Only need the index? Drop the second variable.
	for i := range nums {
		nums[i] *= 2 // ranging gives a COPY of the value, so write via the index
	}
	fmt.Println("doubled:", nums)

	// --- range over a map: order is randomized on purpose ---
	ages := map[string]int{"alice": 30, "bob": 25}
	for k, v := range ages {
		_ = k
		_ = v
	}
	fmt.Println("map len:", len(ages))

	// --- range over a string yields runes (code points), not bytes ---
	for i, r := range "héllo" {
		fmt.Printf("%d:%c ", i, r) // note the index jumps 1->3, é is 2 bytes
	}
	fmt.Println()

	// --- range over an integer (Go 1.22+): repeat N times ---
	for i := range 3 {
		fmt.Print("tick", i, " ")
	}
	fmt.Println()

	// --- continue skips the rest of this iteration ---
	for i := range 6 {
		if i%2 == 0 {
			continue
		}
		fmt.Print(i, " ")
	}
	fmt.Println()

	// --- Labels: break/continue an OUTER loop ---
	grid := [][]int{{1, 2}, {3, 4}, {5, 6}}
outer:
	for _, row := range grid {
		for _, v := range row {
			if v == 4 {
				fmt.Println("found 4, leaving both loops")
				break outer // plain `break` would only exit the inner loop
			}
		}
	}

	// --- range over a function: iterators (Go 1.23+) ---
	for v := range countdown(3) {
		fmt.Print(v, " ")
	}
	fmt.Println()
}

// countdown returns an iterator. yield returns false when the consumer breaks,
// which is your cue to stop and clean up.
func countdown(from int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		for i := from; i > 0; i-- {
			if !yield(i) {
				return
			}
		}
	}
}
