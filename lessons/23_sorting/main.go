// Lesson 23: sorting, searching and the slices/maps packages.
//
// Run: go run ./lessons/23_sorting
package main

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

type Employee struct {
	Name   string
	Dept   string
	Salary int
}

func main() {
	// --- slices.Sort: in place, for any ordered type (Go 1.21+) ---
	nums := []int{5, 2, 8, 1, 9}
	slices.Sort(nums)
	fmt.Println("sorted:", nums)

	words := []string{"pear", "Apple", "banana"}
	slices.Sort(words) // byte order: uppercase sorts first
	fmt.Println("strings:", words)

	// Case-insensitive: supply your own comparison.
	slices.SortFunc(words, func(a, b string) int {
		return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	fmt.Println("case-insensitive:", words)

	// --- SortFunc on structs. The comparator returns <0, 0, >0. ---
	emps := []Employee{
		{"Carol", "eng", 95000},
		{"Alice", "eng", 120000},
		{"Bob", "sales", 85000},
		{"Dave", "sales", 85000},
	}

	// By one field.
	slices.SortFunc(emps, func(a, b Employee) int { return cmp.Compare(a.Salary, b.Salary) })
	fmt.Println("by salary:", names(emps))

	// Descending: swap the arguments.
	slices.SortFunc(emps, func(a, b Employee) int { return cmp.Compare(b.Salary, a.Salary) })
	fmt.Println("by salary desc:", names(emps))

	// Multi-key: cmp.Or returns the first non-zero comparison (Go 1.22+).
	slices.SortFunc(emps, func(a, b Employee) int {
		return cmp.Or(
			cmp.Compare(a.Dept, b.Dept),     // dept ascending
			cmp.Compare(b.Salary, a.Salary), // then salary descending
			cmp.Compare(a.Name, b.Name),     // then name, as a tiebreak
		)
	})
	fmt.Println("dept, salary desc, name:", names(emps))

	// SortStableFunc preserves the order of equal elements - use it when
	// you sort by one key after already sorting by another.
	slices.SortStableFunc(emps, func(a, b Employee) int { return cmp.Compare(a.Dept, b.Dept) })
	fmt.Println("stable by dept:", names(emps))

	// --- Checking and searching ---
	fmt.Println("is sorted:", slices.IsSorted(nums))
	// BinarySearch needs a SORTED slice; it returns the insertion point too.
	i, found := slices.BinarySearch(nums, 8)
	fmt.Println("binary search 8:", i, found)
	i, found = slices.BinarySearch(nums, 6)
	fmt.Println("missing 6 would insert at:", i, found)

	idx, _ := slices.BinarySearchFunc(emps, "sales", func(e Employee, d string) int {
		return cmp.Compare(e.Dept, d)
	})
	fmt.Println("first sales at index:", idx)

	// --- Other slices helpers that remove boilerplate ---
	s := []int{1, 2, 2, 3, 3, 3}
	fmt.Println("compact (dedupe sorted):", slices.Compact(slices.Clone(s)))
	fmt.Println("equal:", slices.Equal([]int{1, 2}, []int{1, 2}))
	fmt.Println("index of 3:", slices.Index(s, 3))
	fmt.Println("index func >2:", slices.IndexFunc(s, func(v int) bool { return v > 2 }))
	fmt.Println("contains func:", slices.ContainsFunc(s, func(v int) bool { return v == 2 }))
	fmt.Println("concat:", slices.Concat([]int{1}, []int{2, 3}))
	fmt.Println("chunked:", slices.Collect(slices.Chunk([]int{1, 2, 3, 4, 5}, 2)))

	// --- maps + slices iterators (Go 1.23+) ---
	salaries := map[string]int{"eng": 215000, "sales": 170000, "hr": 90000}
	fmt.Println("keys sorted:", slices.Sorted(maps.Keys(salaries)))
	fmt.Println("values sum:", sum(slices.Collect(maps.Values(salaries))))

	// Sort map keys BY their values - the "top N" query.
	byValue := slices.SortedFunc(maps.Keys(salaries), func(a, b string) int {
		return cmp.Compare(salaries[b], salaries[a])
	})
	fmt.Println("depts by spend:", byValue)

	// --- The older sort package: still everywhere in real codebases ---
	legacy := []int{3, 1, 2}
	sort.Ints(legacy)
	fmt.Println("sort.Ints:", legacy)
	sort.Slice(emps, func(i, j int) bool { return emps[i].Name < emps[j].Name })
	fmt.Println("sort.Slice by name:", names(emps))
	fmt.Println("sort.SearchInts:", sort.SearchInts(legacy, 2))
	// sort.Sort + sort.Interface (Len/Less/Swap) is the pre-generics way;
	// see lesson 10. Prefer the slices package in new code.
}

func names(emps []Employee) []string {
	out := make([]string, 0, len(emps))
	for _, e := range emps {
		out = append(out, e.Name)
	}
	return out
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}
