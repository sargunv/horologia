package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
)

// Mode controls how output is rendered.
type Mode int

const (
	ModeTable      Mode = iota // Human-readable tables
	ModeJSON                   // Raw JSON
	ModeJSONSchema             // JSON Schema for the command's output
)

// Printer renders command output in the configured mode.
type Printer struct {
	Mode Mode
	Out  io.Writer
}

// New creates a Printer with the given mode, writing to stdout.
func New(mode Mode) *Printer {
	return &Printer{Mode: mode, Out: os.Stdout}
}

// IsSchemaMode returns true if the printer is in JSON schema mode.
func (p *Printer) IsSchemaMode() bool {
	return p.Mode == ModeJSONSchema
}

// ResourceView describes how to display a single resource.
type ResourceView[T any] struct {
	// Value is the resource to display (a struct with json tags for JSON mode).
	Value T
	// Schema is the JSON Schema object for --json-schema mode.
	Schema any
	// Rows are key-value pairs for table mode.
	Rows []KV
}

// KV is a key-value pair for table display.
type KV struct {
	Key   string
	Value string
}

// ListView describes how to display a list of resources.
type ListView[T any] struct {
	// Items are the resources to display (structs with json tags for JSON mode).
	Items []T
	// NextCursor is the pagination cursor, empty if no more results.
	NextCursor string
	// ItemSchema is the JSON Schema for a single item, for --json-schema mode.
	ItemSchema any
	// Headers are the table column headers.
	Headers []string
	// RowFunc extracts table cell values from an item.
	RowFunc func(T) []string
}

// listEnvelope is the JSON structure for paginated list output.
// wrapListSchema must mirror this structure.
type listEnvelope[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// PrintResource renders a single resource.
// This is a package-level function because Go methods cannot have type parameters.
func PrintResource[T any](p *Printer, v ResourceView[T]) error {
	switch p.Mode {
	case ModeJSON:
		return p.writeJSON(v.Value)
	case ModeJSONSchema:
		return p.writeJSON(v.Schema)
	case ModeTable:
		return p.printKeyValue(v.Rows)
	default:
		return fmt.Errorf("output: unknown mode %d", p.Mode)
	}
}

// PrintList renders a list of resources.
// This is a package-level function because Go methods cannot have type parameters.
func PrintList[T any](p *Printer, v ListView[T]) error {
	switch p.Mode {
	case ModeJSON:
		items := v.Items
		if items == nil {
			items = []T{}
		}
		env := listEnvelope[T]{Items: items}
		if v.NextCursor != "" {
			env.NextCursor = &v.NextCursor
		}
		return p.writeJSON(env)
	case ModeJSONSchema:
		return p.writeJSON(wrapListSchema(v.ItemSchema))
	case ModeTable:
		return p.printTable(
			v.Headers,
			len(v.Items),
			func(i int) []string { return v.RowFunc(v.Items[i]) },
			v.NextCursor,
		)
	default:
		return fmt.Errorf("output: unknown mode %d", p.Mode)
	}
}

func (p *Printer) writeJSON(v any) error {
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	cellStyle   = lipgloss.NewStyle()
	faintStyle  = lipgloss.NewStyle().Faint(true)
)

func (p *Printer) printKeyValue(rows []KV) error {
	maxKeyWidth := 0
	for _, kv := range rows {
		if w := lipgloss.Width(kv.Key); w > maxKeyWidth {
			maxKeyWidth = w
		}
	}
	keyStyle := headerStyle.Width(maxKeyWidth + 1)
	for _, kv := range rows {
		_, _ = lipgloss.Fprintf(p.Out, "%s  %s\n", keyStyle.Render(kv.Key+":"), kv.Value)
	}
	return nil
}

func (p *Printer) printTable(headers []string, count int, rowAt func(int) []string, nextCursor string) error {
	if count == 0 {
		_, _ = lipgloss.Fprintln(p.Out, faintStyle.Render("No results."))
		return nil
	}

	rows := make([][]string, count)
	for i := range rows {
		rows[i] = rowAt(i)
	}

	border := lipgloss.NormalBorder()
	border.MiddleTop = "─"
	border.MiddleBottom = "─"
	border.Middle = "─"

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Width(p.terminalWidth()).
		Border(border).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderHeader(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	_, _ = lipgloss.Fprintln(p.Out, t.String())

	if nextCursor != "" {
		_, _ = lipgloss.Fprintln(p.Out, faintStyle.Render(
			fmt.Sprintf("More results available. Use --cursor '%s'", nextCursor),
		))
	}

	return nil
}

// terminalWidth returns the current terminal width, or 120 as a safe default
// for non-interactive contexts (pipes, CI environments).
func (p *Printer) terminalWidth() int {
	type fder interface{ Fd() uintptr }
	if f, ok := p.Out.(fder); ok {
		if w, _, err := term.GetSize(f.Fd()); err == nil && w > 0 {
			return w
		}
	}
	return 120
}

// wrapListSchema wraps an item JSON Schema into the list envelope schema,
// mirroring the structure of listEnvelope[T].
func wrapListSchema(itemSchema any) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type":  "array",
				"items": itemSchema,
			},
			"nextCursor": map[string]any{
				"type": []string{"string", "null"},
			},
		},
		"required": []string{"items", "nextCursor"},
	}
}
