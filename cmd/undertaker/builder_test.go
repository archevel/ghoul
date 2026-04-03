package main

import (
	"testing"
)

func TestSafeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"math", "math"},
		{"path/filepath", "path_filepath"},
		{"encoding/json", "encoding_json"},
		{"github.com/foo/bar", "github.com_foo_bar"},
	}
	for _, tc := range tests {
		got := safeName(tc.input)
		if got != tc.expected {
			t.Errorf("safeName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"math", true},
		{"path/filepath", true},
		{"encoding/json", true},
		{"net/http", true},
		{"crypto/sha256", true},
		{"github.com/foo/bar", false},
		{"golang.org/x/tools", false},
		{"example.com/pkg", false},
	}
	for _, tc := range tests {
		got := isStdlib(tc.pkg)
		if got != tc.expected {
			t.Errorf("isStdlib(%q) = %v, want %v", tc.pkg, got, tc.expected)
		}
	}
}
