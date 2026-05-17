package errors

// Exit codes use a compact 1-6 scheme rather than BSD sysexits.h. sysexits
// would force TypeInternal and TypeFatal to share EX_SOFTWARE (70), and the
// distinction matters: TypeFatal marks a recovered panic that should page,
// TypeInternal marks an expected-but-failed path. The custom scheme keeps
// those orthogonal and stays easy to remember in shell scripts.
//
// The action string is the remediation hint surfaced to users (via the slog
// "exception" payload and the --format=json error envelope). Co-locating it
// with the type means adding a new error class is a one-place change.
var (
	TypeInvalidInput = typ{"invalid-input", 2, ""}
	TypeNotFound     = typ{"not-found", 3, ""}
	TypeUnsupported  = typ{"unsupported", 4, "Please check the documentation for supported features or raise an issue at https://github.com/signoz/foundry/issues for feature requests."}
	TypeInternal     = typ{"internal", 5, ""}
	TypeFatal        = typ{"fatal", 6, "Please raise an issue at https://github.com/signoz/foundry/issues with the error message and stacktrace."}
)

// Defines custom error types, the process exit code they map to, and the
// remediation hint shown to users.
type typ struct {
	s      string
	code   int
	action string
}

func (t typ) String() string {
	return t.s
}

// ExitCode returns the process exit code associated with this error type.
func (t typ) ExitCode() int {
	return t.code
}

// Action returns the remediation hint for this error type, or an empty string
// if no hint applies.
func (t typ) Action() string {
	return t.action
}
