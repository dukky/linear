package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintJSON prints data as JSON
func PrintJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Table is a simple table formatter
type Table struct {
	writer  *tabwriter.Writer
	headers []string
	rows    [][]string
}

// NewTable creates a new table
func NewTable(headers []string) *Table {
	return &Table{
		headers: headers,
		rows:    [][]string{},
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(row []string) {
	t.rows = append(t.rows, row)
}

// Print prints the table to stdout
func (t *Table) Print() {
	t.PrintTo(os.Stdout)
}

// PrintTo prints the table to the given writer
func (t *Table) PrintTo(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Print headers
	fmt.Fprintln(tw, strings.Join(t.headers, "\t"))

	// Print separator
	separators := make([]string, len(t.headers))
	for i, header := range t.headers {
		separators[i] = strings.Repeat("-", len(header))
	}
	fmt.Fprintln(tw, strings.Join(separators, "\t"))

	// Print rows
	for _, row := range t.rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	tw.Flush()
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

// FormatMultilineString formats a multiline string for table display
func FormatMultilineString(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")
	return TruncateString(s, maxLen)
}
