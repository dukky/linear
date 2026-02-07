package output

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "simple object",
			data: map[string]string{
				"key": "value",
			},
			wantErr: false,
		},
		{
			name:    "array",
			data:    []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "nil",
			data:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PrintJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrintJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_AddRow(t *testing.T) {
	table := NewTable([]string{"Col1", "Col2"})

	row1 := []string{"val1", "val2"}
	table.AddRow(row1)

	if len(table.rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(table.rows))
	}

	row2 := []string{"val3", "val4"}
	table.AddRow(row2)

	if len(table.rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.rows))
	}
}

func TestTable_PrintTo(t *testing.T) {
	table := NewTable([]string{"ID", "NAME"})
	table.AddRow([]string{"1", "Test"})
	table.AddRow([]string{"2", "Example"})

	var buf bytes.Buffer
	table.PrintTo(&buf)

	output := buf.String()

	// Check that headers are present
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") {
		t.Error("Output missing headers")
	}

	// Check that data is present
	if !strings.Contains(output, "Test") || !strings.Contains(output, "Example") {
		t.Error("Output missing row data")
	}

	// Check that separator line exists
	if !strings.Contains(output, "--") {
		t.Error("Output missing separator line")
	}
}

func TestTable_EmptyTable(t *testing.T) {
	table := NewTable([]string{"Col1", "Col2"})

	var buf bytes.Buffer
	table.PrintTo(&buf)

	output := buf.String()

	// Should still print headers and separator
	if !strings.Contains(output, "Col1") || !strings.Contains(output, "Col2") {
		t.Error("Empty table should still print headers")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exactly max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "longer than max",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "very short max",
			input:  "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "max of 3",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "zero max length",
			input:  "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "negative max length",
			input:  "hello",
			maxLen: -2,
			want:   "",
		},
		{
			name:   "unicode emoji truncation",
			input:  "hello🙂world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "unicode accented truncation",
			input:  "café résumé",
			maxLen: 8,
			want:   "café...",
		},
		{
			name:   "unicode cjk truncation",
			input:  "多言語サポート対応",
			maxLen: 6,
			want:   "多言語...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString() = %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("TruncateString() produced invalid UTF-8: %q", got)
			}
		})
	}
}

func TestFormatMultilineString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "single line",
			input:  "hello world",
			maxLen: 20,
			want:   "hello world",
		},
		{
			name:   "multiline with newlines",
			input:  "hello\nworld\ntest",
			maxLen: 50,
			want:   "hello world test",
		},
		{
			name:   "multiline with carriage returns",
			input:  "hello\r\nworld",
			maxLen: 50,
			want:   "hello world",
		},
		{
			name:   "multiple spaces collapsed",
			input:  "hello    world",
			maxLen: 50,
			want:   "hello world",
		},
		{
			name:   "multiline and truncate",
			input:  "hello\nworld\nthis is a long text",
			maxLen: 15,
			want:   "hello world ...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "only whitespace",
			input:  "   \n\n   ",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "unicode multiline and truncate",
			input:  "hello\n🙂\n世界 and more text",
			maxLen: 10,
			want:   "hello 🙂...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMultilineString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("FormatMultilineString() = %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("FormatMultilineString() produced invalid UTF-8: %q", got)
			}
		})
	}
}

func TestJSONOutputFormat(t *testing.T) {
	// Test that PrintJSON produces valid JSON
	data := map[string]interface{}{
		"name":  "test",
		"value": 123,
		"items": []string{"a", "b"},
	}

	// Capture output (we can't easily capture stdout, so we'll just verify it doesn't error)
	err := PrintJSON(data)
	if err != nil {
		t.Errorf("PrintJSON should not error on valid data: %v", err)
	}
}

func TestTable_ConsistentOutput(t *testing.T) {
	// Test that the same table produces the same output
	table1 := NewTable([]string{"A", "B"})
	table1.AddRow([]string{"1", "2"})

	table2 := NewTable([]string{"A", "B"})
	table2.AddRow([]string{"1", "2"})

	var buf1, buf2 bytes.Buffer
	table1.PrintTo(&buf1)
	table2.PrintTo(&buf2)

	if buf1.String() != buf2.String() {
		t.Error("Same table data should produce identical output")
	}
}
