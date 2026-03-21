package instrumentation

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
)

const (
	timeFormat       string = "2006-01-02 15:04:05 -07:00"
	moduleName       string = "github.com/signoz/foundry/"
	totalLevelSpaces int    = 6 // total spaces for the level key
)

type PrettyHandler struct {
	out    io.Writer
	opts   Options
	goas   []groupOrAttrs
	mu     *sync.Mutex
	colors *levelColors
}

// levelColors holds pre-allocated color sprint functions for each log level.
type levelColors struct {
	info  func(a ...interface{}) string
	warn  func(a ...interface{}) string
	err   func(a ...interface{}) string
	debug func(a ...interface{}) string
	dim   func(a ...interface{}) string
}

func newLevelColors() *levelColors {
	return &levelColors{
		info:  color.New(color.FgCyan).SprintFunc(),
		warn:  color.New(color.FgYellow).SprintFunc(),
		err:   color.New(color.FgRed, color.Bold).SprintFunc(),
		debug: color.New(color.FgHiBlack).SprintFunc(),
		dim:   color.New(color.FgHiBlack).SprintFunc(),
	}
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler

	// AddSource reports whether to add the source code location of the
	// log statement to the output.
	AddSource bool

	// Debug controls whether to show timestamps and source location.
	// When false (compact mode), only level + message + attrs are shown.
	Debug bool

	// ColorEnabled controls whether ANSI color codes are added to output.
	ColorEnabled bool
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

	h := &PrettyHandler{
		out:  out,
		opts: *opts,
		mu:   &sync.Mutex{},
	}

	if opts.ColorEnabled {
		h.colors = newLevelColors()
	}

	return h
}

func (handler *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= handler.opts.Level.Level()
}

func (handler *PrettyHandler) Handle(ctx context.Context, record slog.Record) error {
	buf := NewBuffer()
	defer buf.Free()

	if handler.opts.Debug {
		// Debug mode: full format with timestamp and source
		// write the time attribute
		buf = handler.appendAttr(buf, slog.Time(slog.TimeKey, record.Time), false, false, true, '|')

		// write the level attribute
		buf = handler.appendLevel(buf, record.Level, true, '|')

		// write the source attribute
		buf = handler.appendAttr(buf, slog.Any(slog.SourceKey, record.Source()), false, true, true, '-')

		// write the message attribute
		buf = handler.appendAttr(buf, slog.String(slog.MessageKey, record.Message), false, true, false, 0)
	} else {
		// Compact mode: level + message only
		buf = handler.appendLevel(buf, record.Level, false, 0)

		// write the message attribute
		buf = handler.appendAttr(buf, slog.String(slog.MessageKey, record.Message), false, true, false, 0)
	}

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

// appendLevel writes the level string with padding and optional color.
func (handler *PrettyHandler) appendLevel(buf *Buffer, level slog.Level, rightSpace bool, sep byte) *Buffer {
	_ = buf.WriteByte(' ')

	levelStr := level.String()
	spaces := max(totalLevelSpaces-len(levelStr), 0)

	if handler.colors != nil {
		var colored string
		switch {
		case level >= slog.LevelError:
			colored = handler.colors.err(levelStr)
		case level >= slog.LevelWarn:
			colored = handler.colors.warn(levelStr)
		case level >= slog.LevelInfo:
			colored = handler.colors.info(levelStr)
		default:
			colored = handler.colors.debug(levelStr)
		}
		_, _ = buf.WriteString(colored)
	} else {
		_, _ = buf.WriteString(levelStr)
	}

	_, _ = buf.Write(bytes.Repeat([]byte(" "), spaces))

	if rightSpace {
		_ = buf.WriteByte(' ')
	}

	if sep != 0 {
		_ = buf.WriteByte(sep)
	}

	return buf
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
			handler.writeKey(buf, attr.Key)
		}
		_, _ = buf.WriteString(attr.Value.Time().Format(timeFormat))

	case slog.KindAny:
		if src, ok := attr.Value.Any().(*slog.Source); ok {
			_, _ = buf.WriteString(strings.TrimPrefix(src.Function, moduleName) + ":" + strconv.Itoa(src.Line))
		}
	default:
		if key {
			handler.writeKey(buf, attr.Key)
		}

		_, _ = buf.WriteString(attr.Value.String())
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

// writeKey writes a key= prefix, with dim color if colors are enabled.
func (handler *PrettyHandler) writeKey(buf *Buffer, k string) {
	if handler.colors != nil {
		_, _ = buf.WriteString(handler.colors.dim(k + "="))
	} else {
		_, _ = buf.WriteString(k + "=")
	}
}
