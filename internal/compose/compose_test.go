package compose

import (
	"testing"
)

func TestParseContainerID(t *testing.T) {
	tests := []struct {
		name   string
		output []byte
		want   string
	}{
		{
			name:   "single line with trailing newline",
			output: []byte("abc123def456\n"),
			want:   "abc123def456",
		},
		{
			name:   "multi-line multiple replicas",
			output: []byte("abc123def456\ndef789abc012\n"),
			want:   "abc123def456",
		},
		{
			name:   "three replica IDs",
			output: []byte("abc123\ndef456\nghi789\n"),
			want:   "abc123",
		},
		{
			name:   "no trailing newline",
			output: []byte("abc123def456"),
			want:   "abc123def456",
		},
		{
			name:   "with surrounding whitespace",
			output: []byte("  abc123def456  \n"),
			want:   "abc123def456",
		},
		{
			name:   "empty output",
			output: []byte(""),
			want:   "",
		},
		{
			name:   "only whitespace",
			output: []byte("  \n  \n"),
			want:   "",
		},
		{
			name:   "multiple lines with empty first line",
			output: []byte("\nabc123def456\n"),
			want:   "abc123def456",
		},
		{
			name:   "single newline only",
			output: []byte("\n"),
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseContainerID(tt.output)
			if got != tt.want {
				t.Errorf("parseContainerID(%q) = %q, want %q", string(tt.output), got, tt.want)
			}
		})
	}
}
