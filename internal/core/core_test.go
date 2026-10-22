package core

import "testing"

func TestSkippedSummary(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, ""},
		{1, "⚠️  1 service skipped (use --show-warnings for details)"},
		{5, "⚠️  5 services skipped (use --show-warnings for details)"},
	}

	for _, tt := range tests {
		if got := skippedSummary(tt.count); got != tt.want {
			t.Errorf("skippedSummary(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}
