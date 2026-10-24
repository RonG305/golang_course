// Lesson 08: pointers and value semantics.
//
// Go has pointers but no pointer arithmetic. The rule to internalise:
// EVERYTHING is passed by value. A pointer is just a value that happens to
// be an address, so passing one lets the callee mutate what it points to.
//
// Run: go run ./lessons/08_pointers
package main

import "fmt"

type counter struct {
	n int
}

// Value receiver: operates on a COPY. The caller sees no change.
func (c counter) incValue() { c.n++ }

// Pointer receiver: operates on the original.
func (c *counter) incPointer() { c.n++ }

func main() {
	// --- & takes an address, * dereferences ---
	x := 10
	p := &x
	fmt.Println("x:", x, "*p:", *p)
	*p = 20
	fmt.Println("x after *p = 20:", x)

	// --- A nil pointer dereference panics. Check before use. ---
	var np *int
	fmt.Println("nil pointer:", np == nil)
	if np != nil {
		fmt.Println(*np)
	}

	// --- Pass by value vs pass by pointer ---
	y := 1
	byValue(y)
	fmt.Println("after byValue:", y)
	byPointer(&y)
	fmt.Println("after byPointer:", y)

	// --- Structs are values too ---
	c := counter{}
	c.incValue()
	fmt.Println("after incValue:", c.n) // still 0
	c.incPointer()                      // Go auto-takes &c for you
	fmt.Println("after incPointer:", c.n)

	// --- new allocates a zeroed value and returns a pointer ---
	q := new(int) // *int pointing at 0
	*q = 5
	fmt.Println("new:", *q)

	// &T{} is the form you will actually use for structs.
	c2 := &counter{n: 7}
	c2.incPointer() // no need to write (*c2).incPointer()
	fmt.Println("c2:", c2.n)

	// --- Slices, maps and channels already hold internal pointers, so the
	//     callee can mutate their CONTENTS without a pointer parameter ---
	s := []int{1, 2, 3}
	mutateSlice(s)
	fmt.Println("slice contents changed:", s)

	// ...but reassigning or appending inside the callee does NOT reach you,
	// because the slice HEADER itself was copied.
	appendSlice(s)
	fmt.Println("slice header unchanged:", s, len(s))

	// --- Pointers to express "optional" / "not set" ---
	type config struct {
		Retries *int // nil means "use the default", 0 means "explicitly zero"
	}
	cfg := config{}
	fmt.Println("retries set:", cfg.Retries != nil)
	cfg.Retries = ptr(0)
	fmt.Println("retries now:", *cfg.Retries)

	// --- Identity: two pointers are == when they point at the same thing ---
	a, bb := &counter{1}, &counter{1}
	fmt.Println("same pointer:", a == bb, "same value:", *a == *bb)

	// --- Escape analysis: returning a pointer to a local is SAFE in Go.
	//     The compiler moves it to the heap. No dangling pointers. ---
	fmt.Println("escaped:", newCounter(42).n)
}

func byValue(v int)    { v = 99 }
func byPointer(v *int) { *v = 99 }

func mutateSlice(s []int) { s[0] = 100 }
func appendSlice(s []int) { s = append(s, 4); _ = s }

func ptr[T any](v T) *T { return &v } // handy generic helper

func newCounter(n int) *counter {
	c := counter{n: n}
	return &c
}
