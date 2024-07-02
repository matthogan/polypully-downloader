package errors

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Msg: "validation error"}
	if err.Error() != "validation error" {
		t.Errorf("Expected 'validation error', got '%s'", err.Error())
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &ValidationError{Err: inner}
	if !errors.Is(err, inner) {
		t.Errorf("Expected inner error, got '%v'", err)
	}
}
