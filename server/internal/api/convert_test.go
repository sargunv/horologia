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

func TestConvert_canonicalizeRelation(t *testing.T) {
	tests := []struct {
		name       string
		kind       apigen.TaskRelationKind
		sourceID   int64
		targetID   int64
		wantKind   string
		wantSource int64
		wantTarget int64
	}{
		{
			name:     "parent_of: no flip",
			kind:     apigen.TaskRelationKindParentOf,
			sourceID: 1, targetID: 2,
			wantKind: "parent", wantSource: 1, wantTarget: 2,
		},
		{
			name:     "child_of: flips source and target",
			kind:     apigen.TaskRelationKindChildOf,
			sourceID: 1, targetID: 2,
			wantKind: "parent", wantSource: 2, wantTarget: 1,
		},
		{
			name:     "blocks: no flip",
			kind:     apigen.TaskRelationKindBlocks,
			sourceID: 3, targetID: 4,
			wantKind: "blocks", wantSource: 3, wantTarget: 4,
		},
		{
			name:     "blocked_by: flips source and target",
			kind:     apigen.TaskRelationKindBlockedBy,
			sourceID: 3, targetID: 4,
			wantKind: "blocks", wantSource: 4, wantTarget: 3,
		},
		{
			name:     "relates_to: already ordered (low, high)",
			kind:     apigen.TaskRelationKindRelatesTo,
			sourceID: 5, targetID: 10,
			wantKind: "relates_to", wantSource: 5, wantTarget: 10,
		},
		{
			name:     "relates_to: reorders (high, low) to (low, high)",
			kind:     apigen.TaskRelationKindRelatesTo,
			sourceID: 10, targetID: 5,
			wantKind: "relates_to", wantSource: 5, wantTarget: 10,
		},
		{
			name:     "duplicates: already ordered (low, high)",
			kind:     apigen.TaskRelationKindDuplicates,
			sourceID: 7, targetID: 8,
			wantKind: "duplicates", wantSource: 7, wantTarget: 8,
		},
		{
			name:     "duplicates: reorders (high, low) to (low, high)",
			kind:     apigen.TaskRelationKindDuplicates,
			sourceID: 8, targetID: 7,
			wantKind: "duplicates", wantSource: 7, wantTarget: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotSource, gotTarget := canonicalizeRelation(tt.kind, tt.sourceID, tt.targetID)
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
			rel:           taskRelationRow{SourceTaskID: 1, TargetTaskID: 2, Kind: "parent", CreatedAt: ts},
			perspectiveID: 1,
			wantKind:      apigen.TaskRelationKindParentOf,
			wantTaskID:    "T2",
		},
		{
			name:          "parent: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 1, TargetTaskID: 2, Kind: "parent", CreatedAt: ts},
			perspectiveID: 2,
			wantKind:      apigen.TaskRelationKindChildOf,
			wantTaskID:    "T1",
		},
		{
			name:          "blocks: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 3, TargetTaskID: 4, Kind: "blocks", CreatedAt: ts},
			perspectiveID: 3,
			wantKind:      apigen.TaskRelationKindBlocks,
			wantTaskID:    "T4",
		},
		{
			name:          "blocks: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 3, TargetTaskID: 4, Kind: "blocks", CreatedAt: ts},
			perspectiveID: 4,
			wantKind:      apigen.TaskRelationKindBlockedBy,
			wantTaskID:    "T3",
		},
		{
			name:          "relates_to: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 5, TargetTaskID: 6, Kind: "relates_to", CreatedAt: ts},
			perspectiveID: 5,
			wantKind:      apigen.TaskRelationKindRelatesTo,
			wantTaskID:    "T6",
		},
		{
			name:          "relates_to: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 5, TargetTaskID: 6, Kind: "relates_to", CreatedAt: ts},
			perspectiveID: 6,
			wantKind:      apigen.TaskRelationKindRelatesTo,
			wantTaskID:    "T5",
		},
		{
			name:          "duplicates: from source perspective",
			rel:           taskRelationRow{SourceTaskID: 7, TargetTaskID: 8, Kind: "duplicates", CreatedAt: ts},
			perspectiveID: 7,
			wantKind:      apigen.TaskRelationKindDuplicates,
			wantTaskID:    "T8",
		},
		{
			name:          "duplicates: from target perspective",
			rel:           taskRelationRow{SourceTaskID: 7, TargetTaskID: 8, Kind: "duplicates", CreatedAt: ts},
			perspectiveID: 8,
			wantKind:      apigen.TaskRelationKindDuplicates,
			wantTaskID:    "T7",
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
		_, err := relationFromDB(taskRelationRow{Kind: "unknown"}, 1)
		if err == nil {
			t.Error("expected error for unknown kind, got nil")
		}
	})
}

func TestConvert_paginate(t *testing.T) {
	identity := func(v int) (*int, error) { return &v, nil }
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
