package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors are part of the domain's public contract. Outer layers map
// them to transport-specific responses (e.g. ErrNotFound -> HTTP 404) with
// errors.Is, so the domain never needs to know about status codes.
var (
	ErrNotFound    = errors.New("task not found")
	ErrAlreadyDone = errors.New("task is already done")
	ErrNotDone     = errors.New("task is not done")
)

// FieldError is one validation problem.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FieldErrors is a collection of them, and is itself an error.
type FieldErrors []FieldError

func (fe FieldErrors) Add(field, msg string) FieldErrors {
	return append(fe, FieldError{Field: field, Message: msg})
}

func (fe FieldErrors) Error() string {
	parts := make([]string, 0, len(fe))
	for _, e := range fe {
		parts = append(parts, fmt.Sprintf("%s %s", e.Field, e.Message))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}
