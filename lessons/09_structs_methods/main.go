// Lesson 09: structs, methods, and composition via embedding.
//
// Run: go run ./lessons/09_structs_methods
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Exported fields start with a capital letter: that is Go's access control.
// Tags (the backtick string) are metadata read by packages like encoding/json.
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	password  string    `json:"-"` // unexported: invisible outside this package
	CreatedAt time.Time `json:"created_at"`
}

// A constructor function. Go has no constructors, so NewX is the convention -
// use it to enforce invariants and set defaults.
func NewUser(id int, name string) *User {
	return &User{ID: id, Name: name, CreatedAt: time.Now()}
}

// Value receiver: reads only, gets a copy. Cheap for small structs.
func (u User) Display() string {
	return fmt.Sprintf("#%d %s", u.ID, u.Name)
}

// Pointer receiver: mutates the original.
func (u *User) Rename(name string) { u.Name = name }

// String() makes the type satisfy fmt.Stringer, so fmt uses it automatically.
func (u User) String() string { return u.Display() }

// --- Embedding: Go's answer to inheritance ---
// An embedded field's methods and fields are PROMOTED to the outer type.
type Base struct {
	ID        int
	CreatedAt time.Time
}

func (b Base) Age() time.Duration { return time.Since(b.CreatedAt) }

type Admin struct {
	Base        // embedded, no field name
	Permissions []string
}

// Overriding: a method on the outer type shadows the promoted one.
type LoudAdmin struct {
	Admin
}

func (l LoudAdmin) Describe() string {
	return strings.ToUpper(fmt.Sprintf("admin %d with %d perms", l.ID, len(l.Permissions)))
}

func main() {
	// --- Creating structs ---
	u1 := User{ID: 1, Name: "Alice"}                // field names: always prefer this
	u2 := User{2, "Bob", "", "secret", time.Time{}} // positional: brittle, avoid
	u3 := NewUser(3, "Carol")
	var u4 User // zero value, immediately usable

	fmt.Println(u1, u2, u3, u4.ID)

	// --- Access and mutation ---
	u1.Email = "alice@example.com"
	u1.Rename("Alice Smith") // Go takes &u1 automatically
	fmt.Println("renamed:", u1, u1.Email)

	// --- Structs are COPIED on assignment and on function calls ---
	copyOfU1 := u1
	copyOfU1.Name = "Changed"
	fmt.Println("original safe:", u1.Name, "| copy:", copyOfU1.Name)

	// Pass a pointer when you need to mutate, or when the struct is large.
	promote(&u1)
	fmt.Println("after promote:", u1.Name)

	// --- Comparability: structs are == if all their fields are comparable ---
	type point struct{ X, Y int }
	fmt.Println("comparable:", point{1, 2} == point{1, 2})
	// Structs containing slices/maps/functions are NOT comparable with ==.

	// --- Embedding in action ---
	a := Admin{
		Base:        Base{ID: 10, CreatedAt: time.Now().Add(-time.Hour)},
		Permissions: []string{"read", "write"},
	}
	fmt.Println("promoted field:", a.ID) // not a.Base.ID
	fmt.Printf("promoted method: %.0f hours\n", a.Age().Hours())
	fmt.Println("explicit path still works:", a.Base.ID)

	l := LoudAdmin{Admin: a}
	fmt.Println(l.Describe(), "| still reachable:", l.Permissions)

	// --- Anonymous structs: perfect for one-off shapes like API responses ---
	resp := struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}{OK: true}
	out, _ := json.Marshal(resp)
	fmt.Println("anonymous:", string(out))

	// --- Struct tags at work: note password is omitted, Email is omitted when empty ---
	b, _ := json.MarshalIndent(u1, "", "  ")
	fmt.Println(string(b))

	// --- Slices of structs vs slices of pointers ---
	users := []User{{ID: 1, Name: "x"}, {ID: 2, Name: "y"}}
	for i := range users {
		users[i].Name += "!" // index, because the range value is a copy
	}
	fmt.Println("mutated in place:", users)

	ptrs := []*User{{ID: 1, Name: "x"}}
	for _, p := range ptrs {
		p.Name += "?" // pointers: mutating through the copy is fine
	}
	fmt.Println("via pointers:", ptrs[0])
}

func promote(u *User) { u.Name = "Dr. " + u.Name }
