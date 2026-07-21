package msgfmt

import (
	"fmt"
	"testing"
)

// stringerArray mimics uuid.UUID: an array type that renders via String().
type stringerArray [2]byte

func (s stringerArray) String() string { return fmt.Sprintf("%02x-%02x", s[0], s[1]) }

func TestRenderMessage(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		want     string
	}{
		{"single key", "{field} bad", map[string]any{"field": "name"}, "name bad"},
		{"multiple keys", "{a}-{b}", map[string]any{"a": 1, "b": 2}, "1-2"},
		{"missing key left untouched", "{x} here", map[string]any{}, "{x} here"},
		{"nil data", "plain", nil, "plain"},
		{"no placeholders", "plain text", map[string]any{"a": 1}, "plain text"},
		{"non-string value", "id {id}", map[string]any{"id": 42}, "id 42"},
		{"string slice joins", "one of {allowed}", map[string]any{"allowed": []string{"a", "b"}}, "one of a, b"},
		{"any slice joins", "one of {allowed}", map[string]any{"allowed": []any{"a", 1}}, "one of a, 1"},
		{"stringer array is not split", "id {id}", map[string]any{"id": stringerArray{0x11, 0x11}}, "id 11-11"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderMessage(tt.template, tt.data); got != tt.want {
				t.Errorf("RenderMessage(%q, %v) = %q, want %q", tt.template, tt.data, got, tt.want)
			}
		})
	}
}
