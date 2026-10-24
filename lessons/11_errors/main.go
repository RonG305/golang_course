// Lesson 11: errors - how Go handles failure.
//
// error is just an interface: interface { Error() string }. Errors are VALUES,
// so you return them, wrap them, inspect them, and build on them.
//
// Run: go run ./lessons/11_errors
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// --- 1. Sentinel errors: a package-level value callers can compare against ---
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// --- 2. Custom error types: when the caller needs structured detail ---
type ValidationError struct {
	Field string
	Value any
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %v", e.Field, e.Value)
}

// A wrapping error type: Unwrap exposes the cause to errors.Is/As.
type QueryError struct {
	Query string
	Err   error
}

func (e *QueryError) Error() string { return fmt.Sprintf("query %q: %v", e.Query, e.Err) }
func (e *QueryError) Unwrap() error { return e.Err }

type store map[string]string

func (s store) Get(key string) (string, error) {
	v, ok := s[key]
	if !ok {
		// %w WRAPS the error: the text is combined and the chain is preserved.
		return "", fmt.Errorf("get %q: %w", key, ErrNotFound)
	}
	return v, nil
}

func (s store) Put(key, value string) error {
	if key == "" {
		return &ValidationError{Field: "key", Value: key}
	}
	if _, exists := s[key]; exists {
		return fmt.Errorf("put %q: %w", key, ErrConflict)
	}
	s[key] = value
	return nil
}

// A service layer that adds its own context on the way up.
func loadUserAge(s store, key string) (int, error) {
	raw, err := s.Get(key)
	if err != nil {
		return 0, fmt.Errorf("loadUserAge: %w", err) // keep wrapping
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &QueryError{Query: key, Err: err}
	}
	return n, nil
}

func main() {
	s := store{"alice": "30", "bob": "not-a-number"}

	// --- errors.Is: compare against a sentinel ANYWHERE in the chain ---
	_, err := loadUserAge(s, "nobody")
	fmt.Println("err:", err)
	fmt.Println("errors.Is(ErrNotFound):", errors.Is(err, ErrNotFound))
	fmt.Println("plain == fails:", err == ErrNotFound) // why you use errors.Is

	// --- errors.As: extract a concrete error type from the chain ---
	_, err = loadUserAge(s, "bob")
	fmt.Println("err:", err)
	var qe *QueryError
	if errors.As(err, &qe) {
		fmt.Println("extracted QueryError on query:", qe.Query)
	}
	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		fmt.Println("root cause was a NumError on input:", numErr.Num)
	}

	// --- Custom error type returned directly ---
	if err := s.Put("", "x"); err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			fmt.Println("validation failed on field:", ve.Field)
		}
	}

	// --- errors.Unwrap walks one level at a time ---
	e := fmt.Errorf("layer3: %w", fmt.Errorf("layer2: %w", ErrConflict))
	for e != nil {
		fmt.Println("chain:", e)
		e = errors.Unwrap(e)
	}

	// --- Joining multiple errors (Go 1.20+): validation, batch jobs ---
	joined := errors.Join(
		&ValidationError{Field: "name", Value: ""},
		&ValidationError{Field: "age", Value: -1},
	)
	fmt.Println("joined:\n" + joined.Error())
	fmt.Println("still matchable:", errors.As(joined, new(*ValidationError)))

	// --- Sentinel vs wrapped: use %w to wrap, %v to flatten ---
	opaque := fmt.Errorf("flattened: %v", ErrNotFound) // chain is BROKEN
	fmt.Println("flattened is matchable:", errors.Is(opaque, ErrNotFound))

	// --- panic/recover: for programmer bugs, NOT for normal error flow ---
	fmt.Println("safe divide:", safeDivide(10, 2))
	fmt.Println("recovered divide:", safeDivide(10, 0))

	// --- Deciding how to fail ---
	// Library code: return an error.
	// main(): log it and exit non-zero.
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		// os.Exit(1) - commented so the lesson finishes normally
	}
	fmt.Println("done")
}

// safeDivide shows recover turning a panic into an error. Note the named
// return value: that is what lets the deferred func change what is returned.
func safeDivide(a, b int) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("recovered from: %v", r)
		}
	}()
	return strconv.Itoa(a / b) // panics when b == 0
}

func run() error { return fmt.Errorf("startup: %w", ErrNotFound) }
