package instrumentation

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	timeFormat       string = "2006-01-02 15:04:05 -07:00"
	moduleName       string = "github.com/signoz/foundry/"
	totalLevelSpaces int    = 6 // total spaces for the level key
)

type PrettyHandler struct {
	out  io.Writer
	opts Options
	goas []groupOrAttrs
	mu   *sync.Mutex
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler

	// AddSource reports whether to add the source code location of the
	// log statement to the output.
	AddSource bool
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

var _ slog.Handler = (*PrettyHandler)(nil)

func newPrettyHandler(out io.Writer, opts *Options) *PrettyHandler {
	if opts == nil {
		opts = &Options{
			Level:     slog.LevelInfo,
			AddSource: true,
		}
	}

	return &PrettyHandler{
		out:  out,
		opts: *opts,
		mu:   &sync.Mutex{},
	}
}

func (handler *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= handler.opts.Level.Level()
}

func (handler *PrettyHandler) Handle(ctx context.Context, record slog.Record) error {
	buf := NewBuffer()
	defer buf.Free()

	// write the time attribute
	buf = handler.appendAttr(buf, slog.Time(slog.TimeKey, record.Time), false, false, true, '|')

	// write the level attribute
	buf = handler.appendAttr(buf, slog.String(slog.LevelKey, record.Level.String()), false, true, true, '|')

	// write the source attribute
	buf = handler.appendAttr(buf, slog.Any(slog.SourceKey, record.Source()), false, true, true, '-')

	// write the message attribute
	buf = handler.appendAttr(buf, slog.String(slog.MessageKey, record.Message), false, true, false, 0)

	goas := handler.goas
	if record.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	for _, goa := range goas {
		if goa.group != "" {
			buf = handler.appendAttr(buf, slog.String("group", goa.group), false, true, false, ':')
		} else {
			for _, attr := range goa.attrs {
				buf = handler.appendAttr(buf, attr, true, true, false, 0)
			}
		}
	}

	record.Attrs(func(attr slog.Attr) bool {
		buf = handler.appendAttr(buf, attr, true, true, false, 0)
		return true
	})

	_ = buf.WriteByte('\n')

	handler.mu.Lock()
	defer handler.mu.Unlock()
	_, err := handler.out.Write(*buf)

	return err
}

func (handler *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return handler
	}

	return handler.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (handler *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return handler
	}

	return handler.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (handler *PrettyHandler) withGroupOrAttrs(goa groupOrAttrs) *PrettyHandler {
	copyOfHandler := *handler
	copyOfHandler.goas = make([]groupOrAttrs, len(handler.goas)+1)
	copy(copyOfHandler.goas, handler.goas)
	copyOfHandler.goas[len(copyOfHandler.goas)-1] = goa
	return &copyOfHandler
}

func (handler *PrettyHandler) appendAttr(buf *Buffer, attr slog.Attr, key bool, leftSpace bool, rightSpace bool, sep byte) *Buffer {
	// Resolve the Attr's value before doing anything else.
	attr.Value = attr.Value.Resolve()

	// Ignore empty Attrs.
	if attr.Equal(slog.Attr{}) {
		return buf
	}

	// Indent 1 space if requested.
	if leftSpace {
		_ = buf.WriteByte(' ')
	}

	// Write the attr.
	switch attr.Value.Kind() {
	case slog.KindTime:
		if attr.Key == slog.TimeKey {
			_, _ = buf.WriteString(attr.Value.Time().Format(timeFormat))
			break
		}

		if key {
			_, _ = buf.WriteString(attr.Key + "=")
		}
		_, _ = buf.WriteString(attr.Value.Time().Format(timeFormat))

	case slog.KindAny:
		if src, ok := attr.Value.Any().(*slog.Source); ok {
			_, _ = buf.WriteString(strings.TrimPrefix(src.Function, moduleName) + ":" + strconv.Itoa(src.Line))
		}
	default:
		if key {
			_, _ = buf.WriteString(attr.Key + "=")
		}

		_, _ = buf.WriteString(attr.Value.String())

		// Add spaces after the level key.
		if attr.Key == slog.LevelKey {
			// The total spaces should be totalLevelSpaces.
			spaces := totalLevelSpaces - len(attr.Value.String())
			if spaces < 0 {
				spaces = 0
			}
			_, _ = buf.Write(bytes.Repeat([]byte(" "), spaces))
		}
	}

	// Add spaces after the attr.
	if rightSpace {
		_ = buf.WriteByte(' ')
	}

	// Add a separator character.
	if sep != 0 {
		_ = buf.WriteByte(sep)
	}

	return buf
}
