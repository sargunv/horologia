package api

import "testing"

func TestAdvanceRotation(t *testing.T) {
	tests := []struct {
		name             string
		pool             []int64
		currentAssignees []int64
		step             int
		want             []int64
	}{
		{
			name:             "empty pool returns nil",
			pool:             []int64{},
			currentAssignees: []int64{1},
			step:             0,
			want:             nil,
		},
		{
			name:             "pool of one always returns that member",
			pool:             []int64{10},
			currentAssignees: []int64{10},
			step:             0,
			want:             []int64{10},
		},
		{
			name:             "advance from first to second",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{1},
			step:             0,
			want:             []int64{2},
		},
		{
			name:             "advance from second to third",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{2},
			step:             0,
			want:             []int64{3},
		},
		{
			name:             "wrap from last to first",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{3},
			step:             0,
			want:             []int64{1},
		},
		{
			name:             "current assignee not in pool uses first",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{99},
			step:             0,
			want:             []int64{1},
		},
		{
			name:             "no current assignees uses first",
			pool:             []int64{1, 2, 3},
			currentAssignees: nil,
			step:             0,
			want:             []int64{1},
		},
		{
			name:             "step advances past normal position",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{1},
			step:             1,
			want:             []int64{3},
		},
		{
			name:             "step wraps around",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{1},
			step:             2,
			want:             []int64{1},
		},
		{
			name:             "step with no current assignee in pool",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{99},
			step:             1,
			want:             []int64{2},
		},
		{
			name:             "multiple current assignees uses first match",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{99, 2, 1},
			step:             0,
			want:             []int64{3},
		},
		{
			name:             "multiple current assignees with step",
			pool:             []int64{1, 2, 3},
			currentAssignees: []int64{99, 2, 1},
			step:             1,
			want:             []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := advanceRotation(tt.pool, tt.currentAssignees, tt.step)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
