package runtime

import (
	"encoding/json"
	"fmt"
)

// PrintJSON writes indented JSON to stdout.
func (a *App) PrintJSON(v any) error {
	enc := json.NewEncoder(a.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Printf writes formatted text to stdout.
func (a *App) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.Stdout, format, args...)
}

// Errorf writes formatted text to stderr.
func (a *App) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.Stderr, format, args...)
}
