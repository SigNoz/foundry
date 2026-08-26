package tooler

import (
	"io"
	"os"
)

// Settings' zero value streams to stderr.
type Settings struct {
	sink io.Writer
}

func NewSettings(sink io.Writer) Settings {
	return Settings{sink: sink}
}

// Sink is where the tool's own output goes; only tests point it elsewhere.
func (s Settings) Sink() io.Writer {
	if s.sink == nil {
		return os.Stderr
	}

	return s.sink
}
