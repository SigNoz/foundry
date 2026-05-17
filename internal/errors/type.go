package errors

// Exit codes use a compact 1-6 scheme rather than BSD sysexits.h. sysexits
// would force TypeInternal and TypeFatal to share EX_SOFTWARE (70), and the
// distinction matters: TypeFatal marks a recovered panic that should page,
// TypeInternal marks an expected-but-failed path. The custom scheme keeps
// those orthogonal and stays easy to remember in shell scripts.
var (
	TypeInvalidInput = typ{"invalid-input", 2}
	TypeNotFound     = typ{"not-found", 3}
	TypeUnsupported  = typ{"unsupported", 4}
	TypeInternal     = typ{"internal", 5}
	TypeFatal        = typ{"fatal", 6}
)

// Defines custom error types and the process exit code they map to.
type typ struct {
	s    string
	code int
}

func (t typ) String() string {
	return t.s
}

// ExitCode returns the process exit code associated with this error type.
func (t typ) ExitCode() int {
	return t.code
}
