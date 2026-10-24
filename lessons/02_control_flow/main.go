// Lesson 02: if/else, switch, and the comma-ok pattern.
//
// Run: go run ./lessons/02_control_flow
package main

import (
	"fmt"
	"strconv"
)

func grade(score int) string {
	// No parentheses around the condition, braces are mandatory.
	if score >= 80 {
		return "A"
	} else if score >= 60 {
		return "B"
	}
	return "C"
}

func main() {
	fmt.Println("75 ->", grade(75))

	// --- if with an initializer: the variable is scoped to the if/else chain ---
	if n, err := strconv.Atoi("42"); err == nil {
		fmt.Println("parsed:", n*2)
	} else {
		fmt.Println("bad number:", err)
	}
	// n and err do not exist out here. This keeps function bodies clean and is
	// the idiomatic way to handle "do a thing, check the error".

	// --- switch: no fallthrough by default, no break needed ---
	day := 6
	switch day {
	case 1, 2, 3, 4, 5: // multiple values per case
		fmt.Println("weekday")
	case 6, 7:
		fmt.Println("weekend")
	default:
		fmt.Println("invalid")
	}

	// --- Expressionless switch: a cleaner if/else-if ladder ---
	temp := 31
	switch {
	case temp < 0:
		fmt.Println("freezing")
	case temp < 20:
		fmt.Println("cold")
	case temp < 30:
		fmt.Println("warm")
	default:
		fmt.Println("hot")
	}

	// --- fallthrough when you really want it ---
	switch 2 {
	case 2:
		fmt.Println("two")
		fallthrough // continues into the next case body unconditionally
	case 3:
		fmt.Println("three (reached via fallthrough)")
	case 4:
		fmt.Println("not reached")
	}

	// --- Type switch: works on interface values (see lesson 10) ---
	for _, v := range []any{1, "hi", 3.5, true, []int{1}} {
		describe(v)
	}
}

func describe(v any) {
	switch t := v.(type) {
	case int:
		fmt.Println("int doubled:", t*2)
	case string:
		fmt.Println("string length:", len(t))
	case float64:
		fmt.Println("float halved:", t/2)
	default:
		fmt.Printf("something else: %T\n", t)
	}
}
