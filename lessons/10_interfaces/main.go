// Lesson 10: interfaces - the heart of Go's design.
//
// An interface is a set of method signatures. Types satisfy it IMPLICITLY -
// there is no "implements" keyword. This means you can write an interface for
// a type you don't own, which is why Go code decouples so easily.
//
// Rule of thumb: accept interfaces, return structs. And keep interfaces small -
// define them where they are CONSUMED, not next to the implementation.
//
// Run: go run ./lessons/10_interfaces
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// A small, focused interface. One method is ideal.
type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rect struct{ W, H float64 }
type Circle struct{ R float64 }

func (r Rect) Area() float64        { return r.W * r.H }
func (r Rect) Perimeter() float64   { return 2 * (r.W + r.H) }
func (c Circle) Area() float64      { return 3.14159 * c.R * c.R }
func (c Circle) Perimeter() float64 { return 2 * 3.14159 * c.R }

// Functions take the interface, so they work with any shape ever written.
func describe(s Shape) string {
	return fmt.Sprintf("%T area=%.2f perim=%.2f", s, s.Area(), s.Perimeter())
}

// --- Dependency injection, the Go way ---
// This service depends on behaviour, not on a concrete type, so tests can pass
// a fake without any mocking framework.
type Notifier interface {
	Notify(msg string) error
}

type EmailNotifier struct{ Addr string }

func (e EmailNotifier) Notify(msg string) error {
	fmt.Printf("  [email->%s] %s\n", e.Addr, msg)
	return nil
}

type FakeNotifier struct{ Sent []string }

func (f *FakeNotifier) Notify(msg string) error {
	f.Sent = append(f.Sent, msg)
	return nil
}

type OrderService struct{ notifier Notifier }

func (s OrderService) Place(id string) error {
	if id == "" {
		return errors.New("empty order id")
	}
	return s.notifier.Notify("order placed: " + id)
}

// --- Implementing a stdlib interface gives you the whole ecosystem ---
// This writer satisfies io.Writer, so it works with fmt.Fprintf, io.Copy, etc.
type upperWriter struct{ w io.Writer }

func (u upperWriter) Write(p []byte) (int, error) {
	n, err := u.w.Write([]byte(strings.ToUpper(string(p))))
	return n, err
}

// sort.Interface: the classic three-method interface.
type byLen []string

func (b byLen) Len() int           { return len(b) }
func (b byLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }
func (b byLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func main() {
	// --- Polymorphism through a slice of interface values ---
	shapes := []Shape{Rect{3, 4}, Circle{2}, Rect{1, 1}}
	for _, s := range shapes {
		fmt.Println(describe(s))
	}

	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	fmt.Printf("total area: %.2f\n", total)

	// --- Type assertion: get the concrete type back out ---
	var s Shape = Circle{5}
	if c, ok := s.(Circle); ok {
		fmt.Println("it is a circle with R =", c.R)
	}
	if _, ok := s.(Rect); !ok {
		fmt.Println("not a Rect")
	}
	// s.(Rect) without the ok would PANIC. Always use the two-value form
	// unless a failure genuinely is a bug.

	// --- Type switch: branch on the dynamic type ---
	for _, sh := range shapes {
		switch v := sh.(type) {
		case Rect:
			fmt.Printf("rect %gx%g\n", v.W, v.H)
		case Circle:
			fmt.Printf("circle r=%g\n", v.R)
		}
	}

	// --- The empty interface: any value. `any` is an alias for interface{}. ---
	var anything any = 42
	anything = "now a string"
	fmt.Println("any holds:", anything)
	// Prefer generics (lesson 12) over `any` when you just need flexibility.

	// --- Injection in practice ---
	fmt.Println("real notifier:")
	real := OrderService{notifier: EmailNotifier{Addr: "ops@example.com"}}
	_ = real.Place("A-1")

	fake := &FakeNotifier{}
	test := OrderService{notifier: fake}
	_ = test.Place("A-2")
	_ = test.Place("A-3")
	fmt.Println("fake captured:", fake.Sent)

	// --- Composing with stdlib interfaces ---
	uw := upperWriter{w: os.Stdout}
	fmt.Fprintln(uw, "written through an io.Writer")

	// --- sort.Interface ---
	words := []string{"banana", "fig", "cherry", "kiwi"}
	sort.Sort(byLen(words))
	fmt.Println("by length:", words)

	// --- Interface embedding: build bigger contracts from smaller ones ---
	// io.ReadWriter is literally: interface { Reader; Writer }
	var rw io.ReadWriter = struct {
		io.Reader
		io.Writer
	}{Reader: strings.NewReader("abc"), Writer: io.Discard}
	buf := make([]byte, 3)
	n, _ := rw.Read(buf)
	fmt.Println("read via embedded interface:", string(buf[:n]))

	// --- THE classic gotcha: a nil pointer inside a non-nil interface ---
	var np *FakeNotifier // nil pointer
	var ni Notifier = np // interface has a TYPE, so it is NOT nil
	fmt.Println("np == nil:", np == nil, "but ni == nil:", ni == nil)
	// Calling ni.Notify here would panic. Never return a typed nil pointer
	// as an interface - return a literal nil instead.

	// --- Checking satisfaction at compile time ---
	var _ Notifier = EmailNotifier{} // fails to compile if the method is missing
	var _ Notifier = (*FakeNotifier)(nil)
	fmt.Println("compile-time checks passed")
}
