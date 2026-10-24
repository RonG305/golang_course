package cart_test // the _test suffix means: test the PUBLIC API only

import (
	"fmt"

	"golangcourse/lessons/24_testing/cart"
)

// A fixed pricer so the example output is deterministic.
type tenPercent struct{}

func (tenPercent) DiscountPercent(string) (int, error) { return 10, nil }

// EXAMPLES are compiled, run as tests AND shown in the docs. The
// "Output:" comment is asserted, so your documentation can never go stale.
func Example() {
	c := cart.New(tenPercent{})
	_ = c.Add(cart.Item{SKU: "book", UnitCents: 2500, Qty: 2})

	total, err := c.Total("SAVE10")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("total: %d cents\n", total)
	// Output: total: 4500 cents
}

// ExampleX_y documents a specific method: this one appears under Cart.Total.
func ExampleCart_Total_noCoupon() {
	c := cart.New(tenPercent{})
	_ = c.Add(cart.Item{SKU: "pen", UnitCents: 150, Qty: 4})
	total, _ := c.Total("")
	fmt.Println(total)
	// Output: 600
}
