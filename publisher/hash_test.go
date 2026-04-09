package publisher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile_MatchesWrangler(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")
	os.WriteFile(path, []byte(`{"type":"object"}`), 0o644)
	hash, err := HashFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 hex chars, got %d: %s", len(hash), hash)
	}
	hash2, err := HashFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != hash2 {
		t.Fatal("same file should produce same hash")
	}
}

func TestHashFile_DifferentExtensionDifferentHash(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte(`{"type":"object"}`)
	jsonPath := filepath.Join(tmpDir, "test.json")
	htmlPath := filepath.Join(tmpDir, "test.html")
	os.WriteFile(jsonPath, content, 0o644)
	os.WriteFile(htmlPath, content, 0o644)
	hashJSON, _ := HashFile(jsonPath)
	hashHTML, _ := HashFile(htmlPath)
	if hashJSON == hashHTML {
		t.Fatal("different extensions should produce different hashes")
	}
}

func TestHashFile_DifferentContentDifferentHash(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "a.json")
	path2 := filepath.Join(tmpDir, "b.json")
	os.WriteFile(path1, []byte(`{"a":1}`), 0o644)
	os.WriteFile(path2, []byte(`{"b":2}`), 0o644)
	hash1, _ := HashFile(path1)
	hash2, _ := HashFile(path2)
	if hash1 == hash2 {
		t.Fatal("different content should produce different hashes")
	}
}
