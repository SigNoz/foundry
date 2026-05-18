package errors

import (
	"fmt"
	"testing"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "Nil_Zero", err: nil, expected: 0},
		{name: "Untyped_One", err: fmt.Errorf("plain error"), expected: 1},
		{name: "InvalidInput_Two", err: Newf(TypeInvalidInput, "bad input"), expected: 2},
		{name: "NotFound_Three", err: Newf(TypeNotFound, "missing"), expected: 3},
		{name: "Unsupported_Four", err: Newf(TypeUnsupported, "no support"), expected: 4},
		{name: "Internal_Five", err: Newf(TypeInternal, "internal"), expected: 5},
		{name: "Fatal_Six", err: Newf(TypeFatal, "fatal"), expected: 6},
		{
			name:     "WrappedWithFmtErrorf_WalksChain",
			err:      fmt.Errorf("outer: %w", Newf(TypeNotFound, "inner")),
			expected: 3,
		},
		{
			name:     "WrappedWithFoundryWrapf_PreservesOuterType",
			err:      Wrapf(Newf(TypeNotFound, "inner"), TypeUnsupported, "outer"),
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}
