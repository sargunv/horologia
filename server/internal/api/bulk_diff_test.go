package api

import (
	"reflect"
	"testing"
)

func TestComputeBulkDiffDeduplicatesResults(t *testing.T) {
	t.Parallel()
	added, removed := computeBulkDiff(
		[]string{"old", "old", "kept"},
		[]string{"new", "new", "kept"},
	)
	if !reflect.DeepEqual(added, []string{"new"}) {
		t.Fatalf("added = %v, want [new]", added)
	}
	if !reflect.DeepEqual(removed, []string{"old"}) {
		t.Fatalf("removed = %v, want [old]", removed)
	}
}
