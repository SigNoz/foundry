package writer

import (
	"encoding/json"
	"io"
)

// WriteOutput writes m's MarshalJSON bytes (with a trailing newline) to w.
// Used for stream payloads that don't have a filesystem path. Each implementer
// owns its own envelope (e.g. errors.Envelope wraps as {"exception": {...}})
// via MarshalJSON, so the writer stays a thin transport.
func WriteOutput(w io.Writer, m json.Marshaler) error {
	data, err := m.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.Write(append(data, '\n'))

	return err
}
