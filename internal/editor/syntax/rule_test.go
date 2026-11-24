package syntax

import (
	"strings"
	"testing"
)

// ruleBefore is rule as it was, copying the line: the oracle.
func ruleBefore(t string) bool {
	s := strings.ReplaceAll(strings.ReplaceAll(t, " ", ""), "\t", "")
	return len(s) >= 3 && strings.Trim(s, s[:1]) == "" && strings.Contains("*-_", s[:1])
}

func TestRuleAsBefore(t *testing.T) {
	parts := []string{"", " ", "\t", "*", "-", "_", "=", "a", "é", "**", "- ", " _"}
	var try func(s string, depth int)
	try = func(s string, depth int) {
		if got, want := rule(s), ruleBefore(s); got != want {
			t.Fatalf("rule(%q) = %v, want %v", s, got, want)
		}
		if depth == 0 {
			return
		}
		for _, p := range parts {
			try(s+p, depth-1)
		}
	}
	try("", 5)
}
