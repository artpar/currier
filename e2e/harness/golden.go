package harness

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// GoldenManager handles golden file operations.
type GoldenManager struct {
	baseDir string
	update  bool
}

// NewGoldenManager creates a golden file manager.
func NewGoldenManager(baseDir string) *GoldenManager {
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	return &GoldenManager{
		baseDir: baseDir,
		update:  update,
	}
}

// Normalize removes dynamic content (timestamps, ports, etc.).
func (g *GoldenManager) Normalize(output string) string {
	// Replace timestamps like "123ms" or "1.5s"
	output = regexp.MustCompile(`\d+(\.\d+)?ms`).ReplaceAllString(output, "XXms")
	output = regexp.MustCompile(`\d+(\.\d+)?s`).ReplaceAllString(output, "XXs")

	// Replace ports in URLs
	output = regexp.MustCompile(`localhost:\d+`).ReplaceAllString(output, "localhost:XXXX")
	output = regexp.MustCompile(`127\.0\.0\.1:\d+`).ReplaceAllString(output, "127.0.0.1:XXXX")

	// Replace UUIDs
	output = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).
		ReplaceAllString(output, "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX")

	return output
}

// Compare compares output against golden file.
func (g *GoldenManager) Compare(t *testing.T, name string, actual string) {
	t.Helper()

	if g.baseDir == "" {
		t.Skip("golden directory not configured")
		return
	}

	goldenPath := filepath.Join(g.baseDir, name+".golden")
	normalized := g.Normalize(actual)

	if g.update {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err != nil {
			t.Fatalf("failed to create golden dir: %v", err)
		}
		err = os.WriteFile(goldenPath, []byte(normalized), 0644)
		if err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v\nActual output:\n%s", goldenPath, err, normalized)
	}

	if string(expected) != normalized {
		t.Errorf("output mismatch for %s\n\nExpected:\n%s\n\nActual:\n%s",
			name, string(expected), normalized)
	}
}

// Update forces an update of the golden file.
func (g *GoldenManager) Update(t *testing.T, name string, content string) {
	t.Helper()

	if g.baseDir == "" {
		t.Fatal("golden directory not configured")
		return
	}

	goldenPath := filepath.Join(g.baseDir, name+".golden")
	normalized := g.Normalize(content)

	err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
	if err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}
	err = os.WriteFile(goldenPath, []byte(normalized), 0644)
	if err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}
	t.Logf("Updated golden file: %s", goldenPath)
}

// IsUpdateMode returns true if golden files should be updated.
func (g *GoldenManager) IsUpdateMode() bool {
	return g.update
}
