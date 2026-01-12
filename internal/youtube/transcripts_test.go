package youtube

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple text", "hello world", "hello world"},
		{"special characters", "file<>:\"/\\|?*name", "file_________name"},
		{"leading spaces", "  filename", "filename"},
		{"trailing spaces", "filename  ", "filename"},
		{"leading dots", "..filename", "filename"},
		{"trailing dots", "filename..", "filename"},
		{"long filename truncated", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"empty after sanitize", "...", ""},
		{"mixed invalid chars", "My Video: Part 1/2 | Q&A?", "My Video_ Part 1_2 _ Q&A_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
