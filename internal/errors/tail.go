package errors

// tailCap bounds what one tail keeps: enough to hold a tool's closing
// diagnostic, small enough to ride inside an error and a log line.
const tailCap = 8 << 10

// Tail keeps the last tailCap bytes written to it: a diagnostic, not a
// transcript. When both of a tool's streams write here, their order is
// whatever the writes were.
//
// Inspired by os/exec's prefixSuffixSaver, which bounds what
// ExitError.Stderr retains for the same reason.
type Tail struct {
	buf []byte
}

func (t *Tail) Write(p []byte) (int, error) {
	if len(p) >= tailCap {
		t.buf = append(t.buf[:0], p[len(p)-tailCap:]...)

		return len(p), nil
	}

	if over := len(t.buf) + len(p) - tailCap; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}

	t.buf = append(t.buf, p...)

	return len(p), nil
}

func (t *Tail) String() string {
	return string(t.buf)
}
