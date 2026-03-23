package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintResourceJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSON, Out: &buf}

	type item struct {
		Name string `json:"name"`
	}

	err := PrintResource(p, ResourceView[item]{
		Value: item{Name: "test"},
	})
	if err != nil {
		t.Fatalf("PrintResource() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("got name = %v, want %q", got["name"], "test")
	}
}

func TestPrintResourceSchema(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSONSchema, Out: &buf}

	schema := map[string]any{"type": "object"}
	err := PrintResource(p, ResourceView[any]{Schema: schema})
	if err != nil {
		t.Fatalf("PrintResource() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["type"] != "object" {
		t.Errorf("got type = %v, want %q", got["type"], "object")
	}
}

func TestPrintListJSON(t *testing.T) {
	t.Run("with items and cursor", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Mode: ModeJSON, Out: &buf}

		type item struct {
			ID string `json:"id"`
		}

		err := PrintList(p, ListView[item]{
			Items:      []item{{ID: "1"}, {ID: "2"}},
			NextCursor: "abc",
		})
		if err != nil {
			t.Fatalf("PrintList() error = %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		items := got["items"].([]any)
		if len(items) != 2 {
			t.Errorf("got %d items, want 2", len(items))
		}
		if got["nextCursor"] != "abc" {
			t.Errorf("got nextCursor = %v, want %q", got["nextCursor"], "abc")
		}
	})

	t.Run("nil items becomes empty array", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Mode: ModeJSON, Out: &buf}

		type item struct{}
		err := PrintList(p, ListView[item]{Items: nil})
		if err != nil {
			t.Fatalf("PrintList() error = %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		items := got["items"].([]any)
		if len(items) != 0 {
			t.Errorf("got %d items, want 0", len(items))
		}
	})

	t.Run("no cursor is null", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Mode: ModeJSON, Out: &buf}

		type item struct{}
		err := PrintList(p, ListView[item]{Items: []item{}})
		if err != nil {
			t.Fatalf("PrintList() error = %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if got["nextCursor"] != nil {
			t.Errorf("got nextCursor = %v, want nil", got["nextCursor"])
		}
	})
}

func TestWrapListSchema(t *testing.T) {
	itemSchema := map[string]any{"type": "string"}
	got := wrapListSchema(itemSchema)

	props := got["properties"].(map[string]any)
	itemsProp := props["items"].(map[string]any)
	if itemsProp["type"] != "array" {
		t.Errorf("items.type = %v, want %q", itemsProp["type"], "array")
	}

	required := got["required"].([]string)
	if len(required) != 2 || required[0] != "items" || required[1] != "nextCursor" {
		t.Errorf("required = %v, want [items, nextCursor]", required)
	}
}

func TestPrintDeletion(t *testing.T) {
	t.Run("table mode prints message", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Mode: ModeTable, Out: &buf}
		p.PrintDeletion("space", "eng")
		if buf.String() != "Deleted space eng\n" {
			t.Errorf("got %q, want %q", buf.String(), "Deleted space eng\n")
		}
	})

	t.Run("json mode prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Mode: ModeJSON, Out: &buf}
		p.PrintDeletion("space", "eng")
		if buf.String() != "" {
			t.Errorf("got %q, want empty", buf.String())
		}
	})
}
