package entities

import (
	"errors"
	"fmt"
	"strings"
)

// Domain errors for the entity module.
var (
	ErrUnknownEntity   = errors.New("unknown entity")
	ErrRecordNotFound  = errors.New("record not found")
	ErrRecordDeleted   = errors.New("record deleted")
	ErrInvalidFilter   = errors.New("invalid filter")
	ErrInvalidSort     = errors.New("invalid sort")
	ErrInvalidPayload  = errors.New("invalid payload")
	ErrValidation      = errors.New("validation failed")
	ErrUnauthenticated = errors.New("unauthenticated")
)

// ValidationErrors aggregates field-level validation failures.
type ValidationErrors []ValidationError

// ValidationError represents a single invalid field.
type ValidationError struct {
	Field   string
	Message string
}

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ErrValidation.Error()
	}
	var parts []string
	for _, e := range v {
		parts = append(parts, fmt.Sprintf("%s: %s", e.Field, e.Message))
	}
	return fmt.Sprintf("%s: %s", ErrValidation.Error(), strings.Join(parts, ", "))
}

func (v ValidationErrors) Is(target error) bool {
	return target == ErrValidation
}
