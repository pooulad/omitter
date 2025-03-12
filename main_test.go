package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// createTempFile is a helper to create a temporary file with the given content.
func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
	return path
}

// TestWalkerWithoutRegex verifies that walker returns a proper mapping when regex is disabled.
func TestWalkerWithoutRegex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testwalker")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create files.
	file1 := createTempFile(t, tempDir, "example_target.txt", "dummy")
	file2 := createTempFile(t, tempDir, "example.txt", "dummy")

	// Call walker with regex disabled (pattern is nil) and str "target".
	pairs, err := walker(tempDir, "target", nil)
	if err != nil {
		t.Fatalf("walker error: %v", err)
	}
	// file1 should be processed because it contains "target".
	if _, ok := pairs[file1]; !ok {
		t.Errorf("expected file %s to be in pairs", file1)
	}
	// file2 should not be processed.
	if _, ok := pairs[file2]; ok {
		t.Errorf("did not expect file %s in pairs", file2)
	}
}

// TestWalkerWithRegex verifies that walker correctly uses a regex pattern.
func TestWalkerWithRegex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testwalker")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create files.
	file1 := createTempFile(t, tempDir, "example_target.txt", "dummy")
	file2 := createTempFile(t, tempDir, "example.txt", "dummy")

	// Compile regex pattern to match "_target" in the file name.
	pattern, err := regexp.Compile("(_target)")
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	// Here the second parameter "target" is still passed,
	// but the searchString function uses the regex if provided.
	pairs, err := walker(tempDir, "target", pattern)
	if err != nil {
		t.Fatalf("walker error: %v", err)
	}

	// file1 should be processed because it matches the regex.
	if _, ok := pairs[file1]; !ok {
		t.Errorf("expected file %s to be in pairs", file1)
	}
	// file2 should not be processed.
	if _, ok := pairs[file2]; ok {
		t.Errorf("did not expect file %s in pairs", file2)
	}

	// Verify that the new file name is as expected.
	expectedNewName := "example.txt" // "example_target.txt" with "_target" removed.
	newPath, ok := pairs[file1]
	if !ok {
		t.Fatalf("file %s not found in pairs", file1)
	}
	if filepath.Base(newPath) != expectedNewName {
		t.Errorf("expected new file name %q, got %q", expectedNewName, filepath.Base(newPath))
	}
}

// TestRename verifies that the rename function renames files as expected.
func TestRename(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testrename")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that should be renamed.
	originalFile := createTempFile(t, tempDir, "example_target.txt", "dummy")

	// Expected new name after removing "target".
	newName := "example_.txt"
	newPath := filepath.Join(tempDir, newName)
	pairs := map[string]string{
		originalFile: newPath,
	}

	// Call rename.
	count, err := rename(pairs)
	if err != nil {
		t.Fatalf("rename error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 file renamed, got %d", count)
	}

	// Verify that the original file no longer exists and the new file does.
	if _, err := os.Stat(originalFile); !os.IsNotExist(err) {
		t.Errorf("expected original file %s to be removed", originalFile)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("expected new file %s to exist, error: %v", newPath, err)
	}
}

// TestSearchString verifies the behavior of searchString.
func TestSearchString(t *testing.T) {
	// When pattern is nil, it should simply return the str parameter.
	result := searchString(nil, "target", "example_target.txt")
	if result != "target" {
		t.Errorf("expected 'target', got '%s'", result)
	}

	// With regex pattern provided.
	pattern, err := regexp.Compile("target")
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}
	result = searchString(pattern, "target", "example_target.txt")
	if result != "target" {
		t.Errorf("expected 'target', got '%s'", result)
	}

	// Test with a non-matching pattern.
	pattern, err = regexp.Compile("nomatch")
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}
	result = searchString(pattern, "nomatch", "example_target.txt")
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

// TestCanProceedYes simulates a "yes" response for canProceed.
func TestCanProceedYes(t *testing.T) {
	// Save original os.Stdin.
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	// Create a pipe and write "y\n".
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.WriteString("y\n")
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdin = r

	if !canProceed() {
		t.Error("expected canProceed() to return true for input 'y'")
	}
}

// TestCanProceedNo simulates a "no" response for canProceed.
func TestCanProceedNo(t *testing.T) {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.WriteString("n\n")
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdin = r

	if canProceed() {
		t.Error("expected canProceed() to return false for input 'n'")
	}
}
