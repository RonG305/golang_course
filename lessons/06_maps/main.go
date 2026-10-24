// Lesson 06: maps - Go's hash table.
//
// Run: go run ./lessons/06_maps
package main

import (
	"fmt"
	"maps"
	"slices"
	"sort"
)

type user struct {
	Name string
	Age  int
}

func main() {
	// --- Literal ---
	ages := map[string]int{
		"Alice": 30,
		"Bob":   25,
	}

	// --- Write, read, delete ---
	ages["Carol"] = 28
	delete(ages, "Bob")
	fmt.Println("len:", len(ages))

	// --- Reading a missing key returns the ZERO value, not an error ---
	fmt.Println("missing key gives:", ages["Dave"]) // 0

	// --- The comma-ok idiom tells you whether the key actually exists ---
	age, ok := ages["Dave"]
	if !ok {
		fmt.Println("Dave not found")
	}
	fmt.Println("age, ok:", age, ok)

	// --- GOTCHA: a nil map can be read but NOT written (panics) ---
	var nilMap map[string]int
	fmt.Println("read from nil map:", nilMap["x"], len(nilMap))
	// nilMap["x"] = 1 // panic: assignment to entry in nil map
	nilMap = make(map[string]int) // always make() before writing
	nilMap["x"] = 1
	fmt.Println("after make:", nilMap)

	// Preallocate when you know roughly how many entries you need.
	sized := make(map[string]int, 100)
	_ = sized

	// --- Iteration order is RANDOM by design. Sort the keys for stable output. ---
	keys := slices.Sorted(maps.Keys(ages)) // Go 1.23+: iterator -> sorted slice
	for _, k := range keys {
		fmt.Printf("%s=%d ", k, ages[k])
	}
	fmt.Println()

	// Pre-1.23 style, still common in codebases:
	keys2 := make([]string, 0, len(ages))
	for k := range ages {
		keys2 = append(keys2, k)
	}
	sort.Strings(keys2)
	fmt.Println("sorted keys:", keys2)

	// --- A set is a map with an empty-struct value (zero bytes of storage) ---
	seen := map[string]struct{}{}
	for _, w := range []string{"go", "rust", "go", "zig"} {
		seen[w] = struct{}{}
	}
	fmt.Println("unique:", len(seen), slices.Sorted(maps.Keys(seen)))

	// --- Counting: the zero value makes this a one-liner ---
	counts := map[rune]int{}
	for _, r := range "hello" {
		counts[r]++ // no "if exists" check needed
	}
	fmt.Println("l appears", counts['l'], "times")

	// --- Map of slices: append to the zero value directly ---
	groups := map[string][]string{}
	groups["fruit"] = append(groups["fruit"], "apple", "pear")
	fmt.Println("groups:", groups)

	// --- Maps of structs: values are NOT addressable ---
	users := map[string]user{"u1": {"Alice", 30}}
	// users["u1"].Age = 31 // compile error
	u := users["u1"] // read, modify, write back
	u.Age = 31
	users["u1"] = u
	fmt.Println("updated:", users["u1"])

	// Use pointer values when you mutate often.
	ptrUsers := map[string]*user{"u1": {"Alice", 30}}
	ptrUsers["u1"].Age = 31 // works
	fmt.Println("via pointer:", *ptrUsers["u1"])

	// --- Maps are reference types: passing one does NOT copy it ---
	mutate(ages)
	fmt.Println("mutated by callee:", ages["Alice"])

	// Clone when the caller must not see your changes.
	safe := maps.Clone(ages)
	safe["Alice"] = 0
	fmt.Println("original intact:", ages["Alice"])

	// --- Equality: compare with maps.Equal, not == ---
	fmt.Println("equal:", maps.Equal(ages, maps.Clone(ages)))
}

func mutate(m map[string]int) { m["Alice"] = 99 }
