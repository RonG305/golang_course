package main

import "fmt"

// `:=` short form only works inside a function.
var appName = "golang-course"

// A const block with iota is how Go does enums.
type Level int

const (
	LevelDebug Level = iota // 0
	LevelInfo               // 1
	LevelWarn               // 2
	LevelError              // 3
)

// Give the enum a String() method and fmt will use it automatically.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	}
	return "UNKNOWN"
}

func main() {
	// --- Declaration styles ---
	var age int = 25 // explicit type
	name := "Ronald" // inferred: string
	price := 99.5    // inferred: float64
	active := true   // inferred: bool

	fmt.Println(appName, age, name, price, active)

	// --- Zero values: every type has one, nothing is ever "undefined" ---
	var (
		i  int
		f  float64
		s  string
		b  bool
		p  *int           // nil
		sl []string       // nil slice (usable with append!)
		m  map[string]int // nil map (readable, NOT writable)
	)
	fmt.Printf("zero values: %d %.1f %q %t %v %v %v\n", i, f, s, b, p, sl, m)

	// --- Conversion is always explicit. Go has no implicit numeric promotion. ---
	total := float64(age) + price // without float64(...) this is a compile error
	fmt.Println("total:", total)

	var big int64 = 1 << 40
	var small int32 = int32(big) // truncates silently - be careful
	fmt.Println("truncated:", small)

	// --- Constants are untyped until used, so they adapt ---
	const Pi = 3.14159
	var radius float32 = 2
	fmt.Printf("area: %.4f\n", Pi*radius*radius) // Pi becomes float32 here

	// --- %v vs %+v vs %#v vs %T: your debugging toolkit ---
	type point struct{ X, Y int }
	pt := point{1, 2}
	fmt.Printf("%v | %+v | %#v | %T\n", pt, pt, pt, pt)

	// --- The enum in action ---
	fmt.Println("level:", LevelWarn, "as int:", int(LevelWarn))

	// --- Shadowing: a classic source of bugs ---
	x := 1
	{
		x := 2 // a NEW variable that hides the outer x
		_ = x
	}
	fmt.Println("x is still:", x)
}
