// Lesson 07: strings, bytes and runes.
//
// A string is an immutable slice of BYTES holding UTF-8 text. len() counts
// bytes, not characters - that distinction causes real bugs.
//
// Run: go run ./lessons/07_strings_runes
package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	s := "héllo, 世界"

	// --- len is BYTES. Count characters by converting to runes. ---
	fmt.Println("bytes:", len(s), "runes:", len([]rune(s)))

	// --- Indexing gives a byte (uint8), not a character ---
	fmt.Printf("s[0]=%d (%c), s[1]=%d\n", s[0], s[0], s[1])

	// --- range decodes UTF-8 and gives you (byteIndex, rune) ---
	for i, r := range s {
		fmt.Printf("%d:%c ", i, r)
	}
	fmt.Println()

	// --- Conversions ---
	bs := []byte(s) // copy to a mutable byte slice
	rs := []rune(s) // copy to a slice of code points
	fmt.Println("roundtrip:", string(bs), string(rs))
	fmt.Println("reversed:", reverse(s)) // must reverse runes, not bytes

	// --- Strings are immutable ---
	// s[0] = 'H' // compile error
	s2 := "H" + s[1:]
	fmt.Println("rebuilt:", s2)

	// --- The strings package: your daily toolbox ---
	t := "  Go is Fun  "
	fmt.Printf("trim=%q upper=%q lower=%q\n", strings.TrimSpace(t), strings.ToUpper(t), strings.ToLower(t))
	fmt.Println("contains:", strings.Contains(t, "is"))
	fmt.Println("hasPrefix:", strings.HasPrefix("golang", "go"))
	fmt.Println("replace:", strings.ReplaceAll("a-b-c", "-", "+"))
	fmt.Println("split:", strings.Split("a,b,c", ","))
	fmt.Println("fields:", strings.Fields(" a  b   c "))
	fmt.Println("join:", strings.Join([]string{"x", "y", "z"}, "/"))
	fmt.Println("repeat:", strings.Repeat("ab", 3))
	fmt.Println("count:", strings.Count("cheese", "e"))
	fmt.Println("index:", strings.Index("chicken", "ken"))
	fmt.Println("cut:", mustCut("key=value", "="))
	fmt.Println("equalfold:", strings.EqualFold("Go", "GO"))

	// --- Building strings: use Builder in loops, NOT += ---
	// += allocates a brand new string every iteration: O(n^2).
	var b strings.Builder
	b.Grow(32) // optional preallocation
	for i := range 5 {
		fmt.Fprintf(&b, "%d,", i) // Builder implements io.Writer
	}
	fmt.Println("built:", strings.TrimSuffix(b.String(), ","))

	// --- Numbers <-> strings: use strconv, not fmt, on hot paths ---
	n, err := strconv.Atoi("42")
	fmt.Println("Atoi:", n, err)
	if _, err := strconv.Atoi("4x2"); err != nil {
		fmt.Println("parse error:", err)
	}
	fmt.Println("Itoa:", strconv.Itoa(99)+"!")
	f, _ := strconv.ParseFloat("3.5", 64)
	fmt.Println("ParseFloat:", f)
	bl, _ := strconv.ParseBool("true")
	fmt.Println("ParseBool:", bl)
	fmt.Println("FormatFloat:", strconv.FormatFloat(3.14159, 'f', 2, 64))
	fmt.Println("Quote:", strconv.Quote("hi\n"))

	// --- Classifying runes ---
	fmt.Println("letters in 'a1!':", countLetters("a1!"))

	// --- Raw string literals keep backslashes and newlines verbatim ---
	fmt.Println(`raw: C:\path\n not escaped`)
}

func reverse(s string) string {
	r := []rune(s) // operate on runes so multi-byte chars survive
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func mustCut(s, sep string) string {
	before, after, found := strings.Cut(s, sep)
	if !found {
		return s
	}
	return before + " -> " + after
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}
