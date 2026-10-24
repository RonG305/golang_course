package cart

import (
	"errors"
	"fmt"
	"testing"
)

// --- A hand-written fake. No mocking framework needed: the interface is tiny. ---
type fakePricer struct {
	discounts map[string]int
	calls     int
}

func (f *fakePricer) DiscountPercent(code string) (int, error) {
	f.calls++
	pct, ok := f.discounts[code]
	if !ok {
		return 0, ErrUnknownCoupo
	}
	return pct, nil
}

func newTestCart() (*Cart, *fakePricer) {
	p := &fakePricer{discounts: map[string]int{"SAVE10": 10, "HALF": 50}}
	return New(p), p
}

// --- TABLE-DRIVEN TESTS: the dominant Go testing style. One test function,
//
//	a slice of cases, t.Run for a named subtest each. ---
func TestCartTotal(t *testing.T) {
	tests := []struct {
		name    string
		items   []Item
		coupon  string
		want    int
		wantErr error
	}{
		{
			name:  "no coupon",
			items: []Item{{SKU: "a", UnitCents: 1000, Qty: 2}},
			want:  2000,
		},
		{
			name:   "ten percent off",
			items:  []Item{{SKU: "a", UnitCents: 1000, Qty: 2}},
			coupon: "SAVE10",
			want:   1800,
		},
		{
			name:   "half off multiple lines",
			items:  []Item{{SKU: "a", UnitCents: 1000, Qty: 1}, {SKU: "b", UnitCents: 500, Qty: 2}},
			coupon: "HALF",
			want:   1000,
		},
		{
			name:    "empty cart",
			items:   nil,
			wantErr: ErrEmptyCart,
		},
		{
			name:    "bad coupon",
			items:   []Item{{SKU: "a", UnitCents: 100, Qty: 1}},
			coupon:  "NOPE",
			wantErr: ErrUnknownCoupo,
		},
	}

	for _, tt := range tests {
		// t.Run gives each case its own name in the output and lets you run
		// just one: go test -run 'TestCartTotal/bad_coupon'
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // safe here: each subtest builds its own cart

			c, _ := newTestCart()
			for _, it := range tt.items {
				if err := c.Add(it); err != nil {
					t.Fatalf("Add(%+v) returned unexpected error: %v", it, err)
				}
			}

			got, err := c.Total(tt.coupon)

			// Compare errors with errors.Is, never with == or string matching.
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Total(%q) error = %v, want %v", tt.coupon, err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return // nothing more to assert on the failure path
			}
			// Good failure messages name the input, the result and the expectation.
			if got != tt.want {
				t.Errorf("Total(%q) = %d, want %d", tt.coupon, got, tt.want)
			}
		})
	}
}

func TestAddValidation(t *testing.T) {
	c, _ := newTestCart()
	err := c.Add(Item{SKU: "x", UnitCents: 100, Qty: 0})
	if !errors.Is(err, ErrInvalidQty) {
		t.Errorf("Add with qty 0: got %v, want ErrInvalidQty", err)
	}
	if c.Len() != 0 {
		t.Errorf("invalid item was added anyway: len = %d", c.Len())
	}
}

func TestAddMergesDuplicateSKUs(t *testing.T) {
	c, _ := newTestCart()
	mustAdd(t, c, Item{SKU: "a", UnitCents: 100, Qty: 1})
	mustAdd(t, c, Item{SKU: "a", UnitCents: 100, Qty: 2})

	if c.Len() != 1 {
		t.Errorf("Len() = %d, want 1 merged line", c.Len())
	}
	if got, want := c.Subtotal(), 300; got != want {
		t.Errorf("Subtotal() = %d, want %d", got, want)
	}
}

// Verifying interactions with a dependency.
func TestCouponIsLookedUpOnce(t *testing.T) {
	c, p := newTestCart()
	mustAdd(t, c, Item{SKU: "a", UnitCents: 100, Qty: 1})
	if _, err := c.Total("SAVE10"); err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 {
		t.Errorf("pricer called %d times, want 1", p.calls)
	}
}

// --- HELPERS: t.Helper() makes failures point at the CALLER's line ---
func mustAdd(t *testing.T, c *Cart, it Item) {
	t.Helper()
	if err := c.Add(it); err != nil {
		t.Fatalf("Add(%+v): %v", it, err)
	}
}

// --- SETUP/TEARDOWN with t.Cleanup: runs even if the test fails ---
func TestWithCleanup(t *testing.T) {
	t.Cleanup(func() { /* close DB, remove temp files, stop servers */ })
	// t.TempDir() is auto-removed; t.Setenv() is auto-restored.
	dir := t.TempDir()
	if dir == "" {
		t.Fatal("expected a temp dir")
	}
}

// --- SKIPPING: keep slow or environment-dependent tests out of the fast path ---
func TestSlowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
	// ... a real DB or network test would live here
}

// --- BENCHMARKS: go test -bench=. -benchmem ---
func BenchmarkSubtotal(b *testing.B) {
	c, _ := newTestCart()
	for i := range 100 {
		_ = c.Add(Item{SKU: fmt.Sprint("sku", i), UnitCents: 199, Qty: 3})
	}
	b.ResetTimer() // exclude the setup above from the measurement
	for b.Loop() { // Go 1.24+ loop form; older code uses `for i := 0; i < b.N; i++`
		_ = c.Subtotal()
	}
}

func BenchmarkAdd(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		c, _ := newTestCart()
		b.StartTimer()
		_ = c.Add(Item{SKU: "a", UnitCents: 100, Qty: 1})
	}
}

// --- FUZZING: go test -fuzz=FuzzAdd -fuzztime=10s
//
//	The fuzzer generates inputs looking for panics and broken invariants. ---
func FuzzAdd(f *testing.F) {
	f.Add("sku", 100, 1) // seed corpus
	f.Add("", 0, -5)
	f.Fuzz(func(t *testing.T, sku string, cents, qty int) {
		c, _ := newTestCart()
		err := c.Add(Item{SKU: sku, UnitCents: cents, Qty: qty})
		// The invariant: Add either succeeds or returns an error. Never panics,
		// and never leaves a negative quantity in the cart.
		if err == nil && qty <= 0 {
			t.Errorf("accepted non-positive qty %d", qty)
		}
	})
}
