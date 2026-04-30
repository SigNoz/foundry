package errors

var (
	TypeInvalidInput typ = typ{"invalid-input"}
	TypeInternal         = typ{"internal"}
	TypeFatal            = typ{"fatal"}
	TypeUnsupported      = typ{"unsupported"}
)

// Defines custom error types.
type typ struct{ s string }

func (t typ) String() string {
	return t.s
}
