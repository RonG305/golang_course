package mathx

// Several files can share one package. Everything in the package sees
// everything else, exported or not - visibility is per PACKAGE, not per file.

// Sum is a variadic convenience wrapper over Add.
func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total = Add(total, n)
	}
	return total
}
