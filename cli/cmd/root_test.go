package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Known command: doctor",
			input:    []string{"brc", "doctor"},
			expected: []string{"brc", "doctor"},
		},
		{
			name:     "Typo command: docter (should not inject)",
			input:    []string{"brc", "docter"},
			expected: []string{"brc", "docter"},
		},
		{
			name:     "Python code with quotes",
			input:    []string{"brc", "print('hello')"},
			expected: []string{"brc", "exec", "print('hello')"},
		},
		{
			name:     "Python file extension",
			input:    []string{"brc", "script.py"},
			expected: []string{"brc", "exec", "script.py"},
		},
		{
			name:     "Global flag with python code",
			input:    []string{"brc", "-s", "1234", "a=1"},
			expected: []string{"brc", "exec", "-s", "1234", "a=1"},
		},
		{
			name:     "No arguments",
			input:    []string{"brc"},
			expected: []string{"brc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %s, got %s", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestPrepareArgs_ExistingFile(t *testing.T) {
	// Create a temporary file without .py extension
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "myscript")
	err := os.WriteFile(tempFile, []byte("print(1)"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := []string{"brc", tempFile}
	expected := []string{"brc", "exec", tempFile}

	result := prepareArgs(input)
	if len(result) != len(expected) || result[1] != "exec" {
		t.Errorf("expected exec to be injected for existing file, got: %v", result)
	}
}
