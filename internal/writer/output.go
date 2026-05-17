package writer

import "io"

// Outputable is implemented by types that can serialize themselves for
// stream-oriented emission via WriteOutput — typed errors today, and
// command result objects (forge / cast / gauge) as they land. Each
// implementer owns its own envelope (e.g. errors.JSONError wraps as
// {"exception": {...}}) so the writer is a thin transport.
type Outputable interface {
	Marshal() ([]byte, error)
}

// WriteOutput writes o's marshaled bytes (with a trailing newline) to w.
// Used for stream payloads that don't have a filesystem path. Marshal and
// write errors propagate to the caller; partial writes are not retried.
func WriteOutput(w io.Writer, o Outputable) error {
	data, err := o.Marshal()
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}
