// Lesson 05: arrays, slices, and the aliasing gotchas that bite everyone.
//
// Run: go run ./lessons/05_arrays_slices
package main

import (
	"fmt"
	"slices"
)

func main() {
	// --- Arrays: fixed length, part of the type, copied on assignment ---
	var arr [3]int
	arr[0] = 1
	brr := arr // a full copy
	brr[0] = 99
	fmt.Println("arrays are values:", arr, brr)
	fmt.Printf("different types: %T vs %T\n", [3]int{}, [4]int{})

	// --- Slices: a view (pointer, len, cap) over a backing array ---
	nums := []int{20, 30, 40, 50, 60, 70}
	nums = append(nums, 80) // append RETURNS a new slice header - always reassign
	fmt.Println(nums, "len:", len(nums), "cap:", cap(nums))

	// --- Slicing: [low:high], low included, high excluded ---
	fmt.Println("nums[1:4]:", nums[1:4])
	fmt.Println("nums[:3]:", nums[:3], "nums[4:]:", nums[4:])

	// --- make: preallocate when you know the size. This matters for performance. ---
	buf := make([]int, 0, 10) // len 0, cap 10 - appends up to 10 never reallocate
	fmt.Println("make:", buf, len(buf), cap(buf))

	withLen := make([]int, 3) // len 3, filled with zero values
	fmt.Println("make with len:", withLen)

	// --- How append grows: capacity roughly doubles, and the backing array moves ---
	g := make([]int, 0)
	prev := cap(g)
	for i := range 9 {
		g = append(g, i)
		if cap(g) != prev {
			fmt.Printf("len %d -> cap %d\n", len(g), cap(g))
			prev = cap(g)
		}
	}

	// --- GOTCHA 1: slices share memory. Writing through one view changes the other. ---
	base := []int{1, 2, 3, 4, 5}
	view := base[1:3]
	view[0] = 999
	fmt.Println("aliased:", base, view)

	// --- GOTCHA 2: append can write into the original's spare capacity ---
	src := []int{1, 2, 3, 4, 5}
	head := src[:2]          // len 2, cap 5 - there is room to the right
	head = append(head, 777) // so this OVERWRITES src[2]
	fmt.Println("clobbered:", src, head)

	// Fix: a full slice expression [low:high:max] caps the capacity, forcing a copy.
	src2 := []int{1, 2, 3, 4, 5}
	safe := src2[:2:2]
	safe = append(safe, 777)
	fmt.Println("safe:", src2, safe)

	// --- copy: explicit duplication ---
	dst := make([]int, len(base))
	n := copy(dst, base)
	dst[0] = -1
	fmt.Println("copied", n, "elements:", base, dst)

	// Or clone in one call.
	fmt.Println("clone:", slices.Clone(base))

	// --- Deleting an element ---
	d := []string{"a", "b", "c", "d"}
	d = slices.Delete(d, 1, 2) // removes index 1
	fmt.Println("deleted:", d)

	// Inserting
	d = slices.Insert(d, 1, "B")
	fmt.Println("inserted:", d)

	// --- The slices package (Go 1.21+) replaces a lot of hand-written loops ---
	s := []int{3, 1, 4, 1, 5, 9, 2, 6}
	slices.Sort(s)
	fmt.Println("sorted:", s)
	fmt.Println("contains 5:", slices.Contains(s, 5))
	fmt.Println("index of 4:", slices.Index(s, 4))
	fmt.Println("max/min:", slices.Max(s), slices.Min(s))
	fmt.Println("reversed:", func() []int { c := slices.Clone(s); slices.Reverse(c); return c }())

	// --- 2D slices: build each row, there is no built-in rectangular slice ---
	grid := make([][]int, 3)
	for i := range grid {
		grid[i] = make([]int, 3)
		grid[i][i] = 1
	}
	fmt.Println("grid:", grid)

	// --- nil slices are fine to use ---
	var nilSlice []int
	fmt.Println("nil slice:", nilSlice == nil, len(nilSlice))
	nilSlice = append(nilSlice, 1) // append allocates for you
	fmt.Println("after append:", nilSlice)
}
