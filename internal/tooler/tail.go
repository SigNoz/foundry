package tooler

// tailCap bounds what one invocation keeps: enough to hold a tool's closing
// diagnostic, small enough to ride inside an error and a log line.
const tailCap = 8 << 10

// tail is a diagnostic, not a transcript: when a mode wires both of the tool's
// streams here, their order is whatever the writes were.
type tail struct {
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
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

func (t *tail) String() string {
	return string(t.buf)
}
