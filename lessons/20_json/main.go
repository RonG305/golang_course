// Lesson 20: JSON - encoding, decoding and struct tags.
//
// Run: go run ./lessons/20_json
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Address struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type User struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email,omitempty"` // omitted when empty
	Tags  []string `json:"tags,omitempty"`
	Age   int      `json:"-"` // never serialised
	Admin bool     `json:"is_admin"`
	// Pointer fields distinguish "absent" from "zero".
	Nickname *string `json:"nickname,omitempty"`
	// Embedded struct: its fields are inlined into the parent object.
	Address
	// A nested struct becomes a nested object.
	Meta struct {
		CreatedAt time.Time `json:"created_at"`
	} `json:"meta"`
	// string option: encode a number as a JSON string (common in JS APIs).
	Balance int64 `json:"balance,string"`
	// Unexported fields are ALWAYS skipped, tag or not.
	secret string
}

// --- Custom marshalling: control the wire format of your own types ---
type Status int

const (
	StatusActive Status = iota
	StatusSuspended
)

func (s Status) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusSuspended:
		return "suspended"
	}
	return "unknown"
}

// MarshalJSON makes Status encode as a readable string instead of a number.
func (s Status) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

func (s *Status) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	switch str {
	case "active":
		*s = StatusActive
	case "suspended":
		*s = StatusSuspended
	default:
		return fmt.Errorf("unknown status %q", str)
	}
	return nil
}

type Account struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
}

func main() {
	// --- Marshal: Go -> JSON ---
	nick := "ali"
	u := User{
		ID: 1, Name: "Alice", Admin: true, Age: 30,
		Nickname: &nick,
		Address:  Address{City: "Nairobi", Country: "KE"},
		Balance:  150,
		secret:   "hidden",
	}
	u.Meta.CreatedAt = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	b, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	fmt.Println("compact:", string(b))

	// Indented, for logs and files.
	pretty, _ := json.MarshalIndent(u, "", "  ")
	fmt.Println(string(pretty))

	// --- Unmarshal: JSON -> Go. Pass a POINTER or nothing happens. ---
	input := `{"id":2,"name":"Bob","email":"bob@example.com","is_admin":false,
	           "city":"Mombasa","country":"KE","balance":"99","tags":["a","b"]}`
	var u2 User
	if err := json.Unmarshal([]byte(input), &u2); err != nil {
		panic(err)
	}
	fmt.Printf("decoded: %+v\n", u2.Name)
	fmt.Println("embedded city:", u2.City, "| balance from string:", u2.Balance)
	fmt.Println("unknown JSON fields are silently IGNORED by default")

	// Reject unknown fields when strictness matters (config files, APIs).
	dec := json.NewDecoder(strings.NewReader(`{"id":3,"nope":1}`))
	dec.DisallowUnknownFields()
	var u3 User
	fmt.Println("strict decode err:", dec.Decode(&u3))

	// --- Matching JSON field names is case-insensitive ---
	var u4 User
	_ = json.Unmarshal([]byte(`{"NAME":"Carol"}`), &u4)
	fmt.Println("case-insensitive:", u4.Name)

	// --- Dynamic JSON: map[string]any when you don't know the shape ---
	var anyMap map[string]any
	_ = json.Unmarshal([]byte(`{"n":42,"s":"x","b":true,"arr":[1,2],"o":{"k":"v"}}`), &anyMap)
	fmt.Printf("n is %T (all JSON numbers decode to float64)\n", anyMap["n"])
	if nested, ok := anyMap["o"].(map[string]any); ok {
		fmt.Println("nested k:", nested["k"])
	}

	// --- json.RawMessage: defer decoding part of a payload ---
	var envelope struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	_ = json.Unmarshal([]byte(`{"type":"user","payload":{"id":9,"name":"Dave"}}`), &envelope)
	if envelope.Type == "user" {
		var inner User
		_ = json.Unmarshal(envelope.Payload, &inner)
		fmt.Println("polymorphic payload:", inner.ID, inner.Name)
	}

	// --- Slices of structs ---
	var users []User
	_ = json.Unmarshal([]byte(`[{"id":1,"name":"A"},{"id":2,"name":"B"}]`), &users)
	fmt.Println("decoded array of", len(users))

	// --- Custom marshaller in action ---
	acc := Account{Name: "svc", Status: StatusSuspended}
	ab, _ := json.Marshal(acc)
	fmt.Println("custom marshal:", string(ab))
	var acc2 Account
	_ = json.Unmarshal([]byte(`{"name":"svc2","status":"active"}`), &acc2)
	fmt.Println("custom unmarshal:", acc2.Status, "==", StatusActive)
	fmt.Println("bad status err:", json.Unmarshal([]byte(`{"status":"x"}`), &acc2))

	// --- Streaming: Encoder/Decoder work on io.Writer/io.Reader.
	//     This is what you use for HTTP bodies and large files. ---
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", " ")
	_ = enc.Encode(map[string]int{"a": 1}) // adds a trailing newline
	fmt.Print("encoded to writer: ", out.String())

	// Decoding a stream of newline-delimited JSON objects (JSONL).
	stream := strings.NewReader("{\"id\":1}\n{\"id\":2}\n{\"id\":3}\n")
	sdec := json.NewDecoder(stream)
	ids := 0
	for sdec.More() {
		var row User
		if err := sdec.Decode(&row); err != nil {
			break
		}
		ids += row.ID
	}
	fmt.Println("streamed id sum:", ids)

	// Straight to stdout - no intermediate buffer.
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"written": "to stdout"})

	// --- Validity check without decoding ---
	fmt.Println("valid:", json.Valid([]byte(`{"a":1}`)), json.Valid([]byte(`{bad}`)))

	// --- Common mistakes ---
	// 1. Forgetting the pointer in Unmarshal -> no data, no error from your code.
	// 2. Unexported fields silently vanish. Export what must serialise.
	// 3. Not handling the error: malformed input leaves the struct half-filled.
	// 4. Comparing float64 from JSON to an int without conversion.
}
