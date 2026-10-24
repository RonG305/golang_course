// Lesson 12: generics (type parameters, Go 1.18+).
//
// Generics let one function or type work over many types with FULL type safety.
// Reach for them when you would otherwise copy-paste a function per type, or
// fall back to `any` plus type assertions.
//
// Run: go run ./lessons/12_generics
package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// --- Basic type parameter. `any` means "no constraint". ---
func Map[T, U any](in []T, f func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

func Filter[T any](in []T, keep func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func Reduce[T, A any](in []T, init A, f func(A, T) A) A {
	acc := init
	for _, v := range in {
		acc = f(acc, v)
	}
	return acc
}

// --- Constraints: an interface used as a type set ---
type Number interface {
	~int | ~int64 | ~float64 // ~ means "any type whose underlying type is this"
}

func Sum[T Number](nums []T) T {
	var total T // zero value of whatever T is
	for _, n := range nums {
		total += n
	}
	return total
}

// cmp.Ordered covers every type usable with < and >.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// --- Generic types ---
type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

// A generic cache: two type parameters, one constrained to be a valid map key.
type Cache[K comparable, V any] struct {
	data map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{data: make(map[K]V)}
}

func (c *Cache[K, V]) Set(k K, v V) { c.data[k] = v }

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.data[k]
	return v, ok
}

// --- Practical helpers you will reuse constantly ---
func Ptr[T any](v T) *T { return &v }

func Deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func GroupBy[T any, K comparable](in []T, key func(T) K) map[K][]T {
	out := map[K][]T{}
	for _, v := range in {
		out[key(v)] = append(out[key(v)], v)
	}
	return out
}

type employee struct {
	Name string
	Dept string
}

func main() {
	nums := []int{1, 2, 3, 4, 5}

	// Type inference means you rarely write the type arguments explicitly.
	fmt.Println("doubled:", Map(nums, func(n int) int { return n * 2 }))
	fmt.Println("to strings:", Map(nums, func(n int) string { return fmt.Sprint("n", n) }))
	fmt.Println("evens:", Filter(nums, func(n int) bool { return n%2 == 0 }))
	fmt.Println("product:", Reduce(nums, 1, func(a, n int) int { return a * n }))
	fmt.Println("joined:", Reduce(nums, "", func(a string, n int) string {
		return a + strings.Repeat("*", n)
	}))

	// Explicit instantiation, when inference cannot help.
	fmt.Println("explicit:", Map[int, float64](nums, func(n int) float64 { return float64(n) / 2 }))

	fmt.Println("sum ints:", Sum(nums))
	fmt.Println("sum floats:", Sum([]float64{1.5, 2.5}))
	fmt.Println("max:", Max(3, 7), Max("apple", "pear"), Max(2.5, 1.5))

	// Generic container, fully type-checked: s.Push("x") would not compile.
	var s Stack[int]
	s.Push(1)
	s.Push(2)
	top, _ := s.Pop()
	fmt.Println("popped:", top, "remaining:", s.Len())

	strs := &Stack[string]{}
	strs.Push("hello")
	v, ok := strs.Pop()
	fmt.Println("string stack:", v, ok)
	_, ok = strs.Pop()
	fmt.Println("empty pop ok:", ok)

	c := NewCache[string, []int]()
	c.Set("primes", []int{2, 3, 5})
	if got, ok := c.Get("primes"); ok {
		fmt.Println("cache:", got)
	}

	// Pointer helpers: very handy for optional JSON/config fields.
	p := Ptr(42)
	fmt.Println("ptr/deref:", *p, Deref(p, 0), Deref[int](nil, -1))

	fmt.Println("keys:", slices.Sorted(slices.Values(Keys(map[string]int{"a": 1, "b": 2}))))

	emps := []employee{{"Alice", "eng"}, {"Bob", "sales"}, {"Carol", "eng"}}
	byDept := GroupBy(emps, func(e employee) string { return e.Dept })
	for _, d := range slices.Sorted(slices.Values(Keys(byDept))) {
		fmt.Println(d, "->", Map(byDept[d], func(e employee) string { return e.Name }))
	}

	// --- When NOT to use generics ---
	// If a plain interface expresses the behaviour you need, use the interface.
	// Generics are about types that vary; interfaces are about behaviour.
	fmt.Println("sorted with stdlib generics:", sortedCopy([]int{3, 1, 2}))
}

func sortedCopy[T cmp.Ordered](in []T) []T {
	out := slices.Clone(in)
	slices.Sort(out)
	return out
}
