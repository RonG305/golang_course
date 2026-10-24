// Package cart is a small, testable domain package: the code under test in
// lesson 24. Note how every dependency it needs is an INTERFACE, which is
// what makes it easy to test without any mocking library.
package cart

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyCart    = errors.New("cart is empty")
	ErrInvalidQty   = errors.New("quantity must be positive")
	ErrUnknownCoupo = errors.New("unknown coupon")
)

type Item struct {
	SKU       string
	UnitCents int
	Qty       int
}

// Pricer looks up a discount percentage for a coupon code. The cart does not
// care whether that is a database, an HTTP call or a map in a test.
type Pricer interface {
	DiscountPercent(code string) (int, error)
}

type Cart struct {
	items  []Item
	pricer Pricer
}

func New(pricer Pricer) *Cart { return &Cart{pricer: pricer} }

func (c *Cart) Add(item Item) error {
	if item.Qty <= 0 {
		return fmt.Errorf("add %s: %w", item.SKU, ErrInvalidQty)
	}
	// Merge duplicates instead of appending a second line.
	for i := range c.items {
		if c.items[i].SKU == item.SKU {
			c.items[i].Qty += item.Qty
			return nil
		}
	}
	c.items = append(c.items, item)
	return nil
}

func (c *Cart) Len() int { return len(c.items) }

// Subtotal returns the total in cents before any discount.
func (c *Cart) Subtotal() int {
	total := 0
	for _, it := range c.items {
		total += it.UnitCents * it.Qty
	}
	return total
}

// Total applies a coupon. An empty code means no discount.
func (c *Cart) Total(coupon string) (int, error) {
	if len(c.items) == 0 {
		return 0, ErrEmptyCart
	}
	sub := c.Subtotal()
	if coupon == "" {
		return sub, nil
	}
	pct, err := c.pricer.DiscountPercent(coupon)
	if err != nil {
		return 0, fmt.Errorf("total: %w", err)
	}
	return sub - (sub * pct / 100), nil
}
