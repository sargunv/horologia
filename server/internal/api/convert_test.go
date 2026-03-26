package api

import (
	"encoding/base64"
	"testing"
	"time"

	apigen "github.com/sargunv/tend/server/api/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func TestConvert_clampLimit(t *testing.T) {
	tests := []struct {
		name string
		opt  apigen.OptInt32
		want int64
	}{
		{"unset defaults to 50", apigen.OptInt32{}, defaultLimit},
		{"zero clamps to default", apigen.NewOptInt32(0), defaultLimit},
		{"negative clamps to default", apigen.NewOptInt32(-5), defaultLimit},
		{"within range", apigen.NewOptInt32(25), 25},
		{"exact lower boundary 1", apigen.NewOptInt32(1), 1},
		{"exact upper boundary 100", apigen.NewOptInt32(100), 100},
		{"above max clamps to 100", apigen.NewOptInt32(200), maxLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampLimit(tt.opt)
			if got != tt.want {
				t.Errorf("clampLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvert_decodeCursorInt64(t *testing.T) {
	encode := func(s string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(s))
	}

	tests := []struct {
		name    string
		opt     apigen.OptString
		want    int64
		wantErr bool
	}{
		{"empty (no cursor)", apigen.OptString{}, 0, false},
		{"valid cursor", apigen.NewOptString(encode("42")), 42, false},
		{"malformed base64", apigen.NewOptString("!!!invalid!!!"), 0, true},
		{"non-numeric decoded value", apigen.NewOptString(encode("abc")), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeCursorInt64(tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeCursorInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeCursorInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvert_formatParseTaskID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		str     string
		wantErr bool
	}{
		{"round-trip", 123, "T123", false},
		{"zero", 0, "T0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatTaskID(tt.id)
			if formatted != tt.str {
				t.Errorf("formatTaskID(%d) = %q, want %q", tt.id, formatted, tt.str)
			}
			parsed, err := parseTaskID(formatted)
			if err != nil {
				t.Fatalf("parseTaskID(%q) unexpected error: %v", formatted, err)
			}
			if parsed != tt.id {
				t.Errorf("parseTaskID(%q) = %d, want %d", formatted, parsed, tt.id)
			}
		})
	}

	errorTests := []struct {
		name string
		str  string
	}{
		{"invalid prefix", "U123"},
		{"no prefix", "123"},
		{"non-numeric", "Tabc"},
		{"empty", ""},
	}
	for _, tt := range errorTests {
		t.Run("error/"+tt.name, func(t *testing.T) {
			_, err := parseTaskID(tt.str)
			if err == nil {
				t.Errorf("parseTaskID(%q) expected error, got nil", tt.str)
			}
		})
	}
}

func TestConvert_formatParseUserID(t *testing.T) {
	tests := []struct {
		name string
		id   int64
		str  string
	}{
		{"round-trip", 456, "U456"},
		{"zero", 0, "U0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatUserID(tt.id)
			if formatted != tt.str {
				t.Errorf("formatUserID(%d) = %q, want %q", tt.id, formatted, tt.str)
			}
			parsed, err := parseUserID(formatted)
			if err != nil {
				t.Fatalf("parseUserID(%q) unexpected error: %v", formatted, err)
			}
			if parsed != tt.id {
				t.Errorf("parseUserID(%q) = %d, want %d", formatted, parsed, tt.id)
			}
		})
	}

	errorTests := []struct {
		name string
		str  string
	}{
		{"invalid prefix", "T123"},
		{"no prefix", "123"},
		{"non-numeric", "Uabc"},
		{"empty", ""},
	}
	for _, tt := range errorTests {
		t.Run("error/"+tt.name, func(t *testing.T) {
			_, err := parseUserID(tt.str)
			if err == nil {
				t.Errorf("parseUserID(%q) expected error, got nil", tt.str)
			}
		})
	}
}

// TestRelationKindExhaustiveness ensures every TaskRelationKind value appears in
// exactly one of directedKindMap or symmetricKinds. Adding a new kind without
// updating these maps will cause this test to fail.
func TestRelationKindExhaustiveness(t *testing.T) {
	for _, k := range apigen.TaskRelationKind("").AllValues() {
		_, inDirected := directedKindMap[k]
		_, inSymmetric := symmetricKinds[types.StoredRelationKind(k)]
		if !inDirected && !inSymmetric {
			t.Errorf("kind %q not found in directedKindMap or symmetricKinds", k)
		}
		if inDirected && inSymmetric {
			t.Errorf("kind %q found in both directedKindMap and symmetricKinds", k)
		}
	}
}

func TestConvert_canonicalizeRelation(t *testing.T) {
	tests := []struct {
		name       string
		kind       apigen.TaskRelationKind
		sourceID   int64
		targetID   int64
		wantKind   types.StoredRelationKind
		wantSource int64
		wantTarget int64
	}{
		{
			name:     "parent_of: no flip",
			kind:     apigen.TaskRelationKindParentOf,
			sourceID: 1, targetID: 2,
			wantKind: types.RelationKindParent, wantSource: 1, wantTarget: 2,
		},
		{
			name:     "child_of: flips source and target",
			kind:     apigen.TaskRelationKindChildOf,
			sourceID: 1, targetID: 2,
			wantKind: types.RelationKindParent, wantSource: 2, wantTarget: 1,
		},
		{
			name:     "blocks: no flip",
			kind:     apigen.TaskRelationKindBlocks,
			sourceID: 3, targetID: 4,
			wantKind: types.RelationKindBlocks, wantSource: 3, wantTarget: 4,
		},
		{
			name:     "blocked_by: flips source and target",
			kind:     apigen.TaskRelationKindBlockedBy,
			sourceID: 3, targetID: 4,
			wantKind: types.RelationKindBlocks, wantSource: 4, wantTarget: 3,
		},
		{
			name:     "relates_to: already ordered (low, high)",
			kind:     apigen.TaskRelationKindRelatesTo,
			sourceID: 5, targetID: 10,
			wantKind: types.RelationKindRelatesTo, wantSource: 5, wantTarget: 10,
		},
		{
			name:     "relates_to: reorders (high, low) to (low, high)",
			kind:     apigen.TaskRelationKindRelatesTo,
			sourceID: 10, targetID: 5,
			wantKind: types.RelationKindRelatesTo, wantSource: 5, wantTarget: 10,
		},
		{
			name:     "duplicates: already ordered (low, high)",
			kind:     apigen.TaskRelationKindDuplicates,
			sourceID: 7, targetID: 8,
			wantKind: types.RelationKindDuplicates, wantSource: 7, wantTarget: 8,
		},
		{
			name:     "duplicates: reorders (high, low) to (low, high)",
			kind:     apigen.TaskRelationKindDuplicates,
			sourceID: 8, targetID: 7,
			wantKind: types.RelationKindDuplicates, wantSource: 7, wantTarget: 8,
		},
		{
			name:     "triggers: no flip",
			kind:     apigen.TaskRelationKindTriggers,
			sourceID: 11, targetID: 12,
			wantKind: types.RelationKindTriggers, wantSource: 11, wantTarget: 12,
		},
		{
			name:     "triggered_by: flips source and target",
			kind:     apigen.TaskRelationKindTriggeredBy,
			sourceID: 11, targetID: 12,
			wantKind: types.RelationKindTriggers, wantSource: 12, wantTarget: 11,
		},
		{
			name:     "spawns: no flip",
			kind:     apigen.TaskRelationKindSpawns,
			sourceID: 9, targetID: 10,
			wantKind: types.RelationKindSpawns, wantSource: 9, wantTarget: 10,
		},
		{
			name:     "spawned_by: flips source and target",
			kind:     apigen.TaskRelationKindSpawnedBy,
			sourceID: 9, targetID: 10,
			wantKind: types.RelationKindSpawns, wantSource: 10, wantTarget: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotSource, gotTarget, err := canonicalizeRelation(tt.kind, tt.sourceID, tt.targetID)
			if err != nil {
				t.Fatalf("canonicalizeRelation() unexpected error: %v", err)
			}
			if gotKind != tt.wantKind {
				t.Errorf("kind = %q, want %q", gotKind, tt.wantKind)
			}
			if gotSource != tt.wantSource {
				t.Errorf("source = %d, want %d", gotSource, tt.wantSource)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("target = %d, want %d", gotTarget, tt.wantTarget)
			}
		})
	}
}

func TestConvert_relationFromDB(t *testing.T) {
	ts := types.EpochSeconds(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name          string
		rel           taskRelationRow
		perspectiveID int64
		wantKind      apigen.TaskRelationKind
		wantTaskID    string
	}{
		{
			name:          "parent: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 1, TargetTaskID: 2, Kind: types.RelationKindParent, CreatedAt: ts},
			perspectiveID: 1,
			wantKind:      apigen.TaskRelationKindParentOf,
			wantTaskID:    "T2",
		},
		{
			name:          "parent: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 1, TargetTaskID: 2, Kind: types.RelationKindParent, CreatedAt: ts},
			perspectiveID: 2,
			wantKind:      apigen.TaskRelationKindChildOf,
			wantTaskID:    "T1",
		},
		{
			name:          "blocks: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 3, TargetTaskID: 4, Kind: types.RelationKindBlocks, CreatedAt: ts},
			perspectiveID: 3,
			wantKind:      apigen.TaskRelationKindBlocks,
			wantTaskID:    "T4",
		},
		{
			name:          "blocks: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 3, TargetTaskID: 4, Kind: types.RelationKindBlocks, CreatedAt: ts},
			perspectiveID: 4,
			wantKind:      apigen.TaskRelationKindBlockedBy,
			wantTaskID:    "T3",
		},
		{
			name:          "relates_to: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 5, TargetTaskID: 6, Kind: types.RelationKindRelatesTo, CreatedAt: ts},
			perspectiveID: 5,
			wantKind:      apigen.TaskRelationKindRelatesTo,
			wantTaskID:    "T6",
		},
		{
			name:          "relates_to: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 5, TargetTaskID: 6, Kind: types.RelationKindRelatesTo, CreatedAt: ts},
			perspectiveID: 6,
			wantKind:      apigen.TaskRelationKindRelatesTo,
			wantTaskID:    "T5",
		},
		{
			name:          "duplicates: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 7, TargetTaskID: 8, Kind: types.RelationKindDuplicates, CreatedAt: ts},
			perspectiveID: 7,
			wantKind:      apigen.TaskRelationKindDuplicates,
			wantTaskID:    "T8",
		},
		{
			name:          "duplicates: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 7, TargetTaskID: 8, Kind: types.RelationKindDuplicates, CreatedAt: ts},
			perspectiveID: 8,
			wantKind:      apigen.TaskRelationKindDuplicates,
			wantTaskID:    "T7",
		},
		{
			name:          "triggers: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 11, TargetTaskID: 12, Kind: types.RelationKindTriggers, CreatedAt: ts},
			perspectiveID: 11,
			wantKind:      apigen.TaskRelationKindTriggers,
			wantTaskID:    "T12",
		},
		{
			name:          "triggers: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 11, TargetTaskID: 12, Kind: types.RelationKindTriggers, CreatedAt: ts},
			perspectiveID: 12,
			wantKind:      apigen.TaskRelationKindTriggeredBy,
			wantTaskID:    "T11",
		},
		{
			name:          "spawns: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 9, TargetTaskID: 10, Kind: types.RelationKindSpawns, CreatedAt: ts},
			perspectiveID: 9,
			wantKind:      apigen.TaskRelationKindSpawns,
			wantTaskID:    "T10",
		},
		{
			name:          "spawns: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 9, TargetTaskID: 10, Kind: types.RelationKindSpawns, CreatedAt: ts},
			perspectiveID: 10,
			wantKind:      apigen.TaskRelationKindSpawnedBy,
			wantTaskID:    "T9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relationFromDB(tt.rel, tt.perspectiveID)
			if err != nil {
				t.Fatalf("relationFromDB() unexpected error: %v", err)
			}
			if got.Kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", got.Kind, tt.wantKind)
			}
			if got.TaskId != tt.wantTaskID {
				t.Errorf("taskId = %q, want %q", got.TaskId, tt.wantTaskID)
			}
		})
	}

	t.Run("unknown kind returns error", func(t *testing.T) {
		_, err := relationFromDB(taskRelationRow{Kind: "unknown_kind"}, 1)
		if err == nil {
			t.Error("expected error for unknown kind, got nil")
		}
	})
}

func TestConvert_paginate(t *testing.T) {
	identity := func(rows []int) ([]int, error) { return rows, nil }
	cursorOf := func(v int) string { return "c" }

	tests := []struct {
		name     string
		rows     []int
		limit    int64
		wantLen  int
		wantNext bool
	}{
		{"empty input", nil, 10, 0, false},
		{"under limit", []int{1, 2}, 10, 2, false},
		{"exactly at limit", []int{1, 2, 3}, 3, 3, false},
		{"over limit (has next page)", []int{1, 2, 3, 4}, 3, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, next, err := paginate(tt.rows, tt.limit, identity, cursorOf)
			if err != nil {
				t.Fatalf("paginate() unexpected error: %v", err)
			}
			if len(items) != tt.wantLen {
				t.Errorf("len(items) = %d, want %d", len(items), tt.wantLen)
			}
			if tt.wantNext && next.Null {
				t.Error("expected next cursor to be set, got null")
			}
			if !tt.wantNext && !next.Null {
				t.Errorf("expected next cursor to be null, got %q", next.Value)
			}
		})
	}
}

// Sync tests: verify that domain enum values match the apigen AllValues() sets.
// If a value is added to the OpenAPI spec but not to internal/types (or vice versa),
// these tests will fail.

func TestEnumSync_RecurrenceType(t *testing.T) {
	assertEnumSync(t, apigen.TaskRecurrenceType("").AllValues(), types.AllRecurrenceTypes())
}

func TestEnumSync_StatusCategory(t *testing.T) {
	assertEnumSync(t, apigen.TaskStatusCategory("").AllValues(), types.AllStatusCategories())
}

func TestEnumSync_StatusInputCategory(t *testing.T) {
	assertEnumSync(t, apigen.TaskStatusInputCategory("").AllValues(), types.AllStatusCategories())
}

func TestEnumSync_SpaceRole(t *testing.T) {
	assertEnumSync(t, apigen.SpaceRole("").AllValues(), types.AllSpaceRoles())
}

func TestEnumSync_AuthTokenKind(t *testing.T) {
	assertEnumSync(t, apigen.AuthTokenKind("").AllValues(), types.AllAuthTokenKinds())
}

func TestEnumSync_StoredRelationKind(t *testing.T) {
	// Every StoredRelationKind must be reachable from at least one API kind
	// (either as a directed kind's storedKind, or as a symmetric kind).
	allStored := types.AllStoredRelationKinds()
	for _, sk := range allStored {
		found := false
		for _, c := range directedKindMap {
			if c.storedKind == sk {
				found = true
				break
			}
		}
		if !found {
			if _, ok := symmetricKinds[sk]; ok {
				found = true
			}
		}
		if !found {
			t.Errorf("stored relation kind %q is not reachable from any API kind", sk)
		}
	}
}

// assertEnumSync checks that two string-backed enum slices contain the same values.
func assertEnumSync[A ~string, B ~string](t *testing.T, apiValues []A, domainValues []B) {
	t.Helper()
	apiSet := make(map[string]struct{}, len(apiValues))
	for _, v := range apiValues {
		apiSet[string(v)] = struct{}{}
	}
	domainSet := make(map[string]struct{}, len(domainValues))
	for _, v := range domainValues {
		domainSet[string(v)] = struct{}{}
	}
	for v := range apiSet {
		if _, ok := domainSet[v]; !ok {
			t.Errorf("apigen has %q but domain types package does not", v)
		}
	}
	for v := range domainSet {
		if _, ok := apiSet[v]; !ok {
			t.Errorf("domain types package has %q but apigen does not", v)
		}
	}
}
