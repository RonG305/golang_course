// Package mathx provides small numeric helpers.
//
// This doc comment is what `go doc` shows. Start it with "Package <name>".
package mathx

// Version is exported: capital letter, visible to importers.
const Version = "1.0.0"

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }

// Mean returns the arithmetic mean, or 0 for an empty slice.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return normalise(sum / float64(len(xs)))
}

// normalise is unexported - lower-case, so it is private to this package.
func normalise(f float64) float64 {
	if f == 0 {
		return 0 // avoid returning -0
	}
	return f
}

// Accumulator keeps a running total. Note that the field is unexported, so
// callers must go through the methods - that is how you protect invariants.
type Accumulator struct {
	total int
}

// NewAccumulator returns a ready-to-use Accumulator.
func NewAccumulator() *Accumulator { return &Accumulator{} }

// Add adds n to the running total.
func (a *Accumulator) Add(n int) { a.total += n }

// Total returns the current total.
func (a *Accumulator) Total() int { return a.total }
