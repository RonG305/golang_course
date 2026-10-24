// Lesson 24: testing. Go ships its test framework in the standard library -
// no assertion libraries required.
//
// The real content of this lesson is in ./cart/:
//
//	cart/cart.go         the code under test
//	cart/cart_test.go    table-driven tests, fakes, benchmarks, a fuzz target
//	cart/example_test.go runnable documentation examples
//
// Run the lesson:   go run  ./lessons/24_testing
// Run the tests:    go test ./lessons/24_testing/...
//
// Run: go run ./lessons/24_testing
package main

import (
	"fmt"

	"golangcourse/lessons/24_testing/cart"
)

type staticPricer map[string]int

func (p staticPricer) DiscountPercent(code string) (int, error) {
	pct, ok := p[code]
	if !ok {
		return 0, cart.ErrUnknownCoupo
	}
	return pct, nil
}

func main() {
	c := cart.New(staticPricer{"SAVE10": 10})
	_ = c.Add(cart.Item{SKU: "book", UnitCents: 2500, Qty: 2})
	_ = c.Add(cart.Item{SKU: "pen", UnitCents: 150, Qty: 4})

	fmt.Println("subtotal:", c.Subtotal())
	total, err := c.Total("SAVE10")
	fmt.Println("total with SAVE10:", total, "err:", err)

	fmt.Println(`
--- The test commands you need ---
  go test ./...                          test everything
  go test -v ./lessons/24_testing/...    verbose: one line per subtest
  go test -run 'TestCartTotal/empty'     run one subtest by name
  go test -race ./...                    with the race detector (do this in CI)
  go test -short ./...                   skip tests guarded by testing.Short()
  go test -count=1 ./...                 bypass the test result cache
  go test -cover ./...                   coverage percentage
  go test -coverprofile=c.out ./... && go tool cover -html=c.out
  go test -bench=. -benchmem ./...       benchmarks with allocation counts
  go test -fuzz=FuzzAdd -fuzztime=10s ./lessons/24_testing/cart

--- Conventions ---
  File name must end in _test.go, and is excluded from normal builds.
  func TestXxx(t *testing.T)        a test
  func BenchmarkXxx(b *testing.B)   a benchmark
  func FuzzXxx(f *testing.F)        a fuzz target
  func ExampleXxx()                 runnable documentation
  package foo       internal test: can reach unexported identifiers
  package foo_test  external test: forces you to use the public API

--- t.Error vs t.Fatal ---
  t.Errorf  records the failure and KEEPS GOING (check several things)
  t.Fatalf  records it and stops this test (when continuing is pointless)

--- How to write tests that stay useful ---
  Table-driven, with a descriptive name per case.
  Assert on behaviour, not on internals.
  Inject dependencies as small interfaces and fake them by hand.
  Make failure messages say: what was called, what you got, what you wanted.
  Keep tests independent - t.Parallel() should always be safe to add.`)
}
