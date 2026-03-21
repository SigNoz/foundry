package ux

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Table renders a simple aligned table.
type Table struct {
	Headers []string
	Rows    [][]string

	bold func(a ...interface{}) string
	dim  func(a ...interface{}) string
}

// NewTable creates a new table with color support.
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		bold:    color.New(color.Bold).SprintFunc(),
		dim:     color.New(color.FgHiBlack).SprintFunc(),
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// Render writes the table to the given writer.
func (t *Table) Render(w io.Writer) {
	if len(t.Headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	fmt.Fprintf(w, "  ")
	for i, h := range t.Headers {
		fmt.Fprintf(w, "%-*s", widths[i]+3, t.bold(h))
	}
	fmt.Fprintln(w)

	// Print separator
	fmt.Fprintf(w, "  ")
	for i := range t.Headers {
		fmt.Fprintf(w, "%s   ", t.dim(strings.Repeat("─", widths[i])))
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range t.Rows {
		fmt.Fprintf(w, "  ")
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(w, "%-*s   ", widths[i], cell)
			}
		}
		fmt.Fprintln(w)
	}
}
