package ux

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/signoz/foundry/internal/instrumentation"
)

// UX provides user-facing output with spinners in TTY mode
// and plain log lines in non-TTY or debug mode.
type UX struct {
	logger  *slog.Logger
	spinner *spinner.Spinner
	isTTY   bool
	debug   bool
	out     io.Writer

	// Color functions for step output.
	green func(a ...interface{}) string
	red   func(a ...interface{}) string
	dim   func(a ...interface{}) string

	stepStart time.Time
}

// New creates a new UX instance. In debug mode, spinners are disabled
// and all output goes through the structured logger.
func New(debug bool) *UX {
	isTTY := isTerminal(os.Stdout)

	u := &UX{
		logger: instrumentation.NewLogger(debug),
		isTTY:  isTTY,
		debug:  debug,
		out:    os.Stdout,
		green:  color.New(color.FgGreen).SprintFunc(),
		red:    color.New(color.FgRed).SprintFunc(),
		dim:    color.New(color.FgHiBlack).SprintFunc(),
	}

	if isTTY && !debug {
		u.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stdout), spinner.WithColor("cyan"))
	}

	return u
}

// Logger returns the underlying slog.Logger for code that needs structured logging.
func (u *UX) Logger() *slog.Logger {
	return u.logger
}

// StartStep begins a new step. In TTY non-debug mode, shows a spinner.
// In other modes, prints a log line.
func (u *UX) StartStep(msg string) {
	u.stepStart = time.Now()

	if u.spinner != nil {
		u.spinner.Suffix = " " + msg + "..."
		u.spinner.Start()
		return
	}

	if u.debug {
		u.logger.Info(msg)
	} else {
		fmt.Fprintf(u.out, "  %s %s...\n", u.dim("●"), msg)
	}
}

// FinishStep completes the current step. If err is nil, shows a green checkmark.
// If err is non-nil, shows a red X.
func (u *UX) FinishStep(msg string, err error) {
	duration := time.Since(u.stepStart)
	durationStr := formatDuration(duration)

	if u.spinner != nil {
		u.spinner.Stop()
	}

	if u.debug {
		if err != nil {
			u.logger.Error(msg, slog.Any("err", err))
		} else {
			u.logger.Info(msg)
		}
		return
	}

	if err != nil {
		fmt.Fprintf(u.out, "  %s %s %s\n", u.red("✗"), msg, u.dim(durationStr))
	} else {
		fmt.Fprintf(u.out, "  %s %s %s\n", u.green("✓"), msg, u.dim(durationStr))
	}
}

// Success prints a green checkmark status line (no duration).
func (u *UX) Success(msg string) {
	if u.debug {
		u.logger.Info(msg)
		return
	}
	fmt.Fprintf(u.out, "  %s %s\n", u.green("✓"), msg)
}

// Header prints a header line for a pipeline stage.
func (u *UX) Header(msg string) {
	if u.debug {
		u.logger.Info(msg)
		return
	}

	fmt.Fprintf(u.out, "\n  %s\n\n", msg)
}

// MissingTool represents a tool that was not found during gauge.
type MissingTool struct {
	Name        string
	InstallHint string
}

// PrintMissingTools prints a formatted list of missing tools with install hints.
func (u *UX) PrintMissingTools(tools []MissingTool) {
	if u.debug {
		for _, t := range tools {
			u.logger.Error("tool not available", slog.String("tool", t.Name), slog.String("install", t.InstallHint))
		}
		return
	}

	fmt.Fprintf(u.out, "\n  %s Missing tools:\n\n", u.red("✗"))
	table := NewTable("Tool", "Install")
	for _, t := range tools {
		table.AddRow(t.Name, t.InstallHint)
	}
	table.Render(u.out)
	fmt.Fprintln(u.out)
}

// PrintFileSummary prints a summary table of written files.
func (u *UX) PrintFileSummary(files []WrittenFile) {
	if u.debug || len(files) == 0 {
		return
	}

	fmt.Fprintln(u.out)
	table := NewTable("File", "Size")
	for _, f := range files {
		table.AddRow(f.Path, formatBytes(f.Size))
	}
	table.Render(u.out)
	fmt.Fprintln(u.out)
}

// WrittenFile represents a file that was written during forge.
type WrittenFile struct {
	Path string
	Size int64
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("(%dms)", d.Milliseconds())
	}
	return fmt.Sprintf("(%.1fs)", d.Seconds())
}
