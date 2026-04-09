package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_CreatesIndexHTML(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "cert-manager.io"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "cert-manager.io", "certificate_v1.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(tmpDir, "monitoring.coreos.com"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "monitoring.coreos.com", "servicemonitor_v1.json"), []byte(`{}`), 0o644)

	err := Generate(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "index.html"))
	if err != nil {
		t.Fatalf("index.html not created: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "cert-manager.io") {
		t.Fatal("index should contain cert-manager.io group")
	}
	if !strings.Contains(html, "monitoring.coreos.com") {
		t.Fatal("index should contain monitoring.coreos.com group")
	}
	if !strings.Contains(html, "certificate_v1.json") {
		t.Fatal("index should link to schema file")
	}
	if !strings.Contains(html, "2 CRD") {
		t.Fatal("index should show total count")
	}
}

func TestGenerate_SkipsMasterStandalone(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "example.io"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "example.io", "test_v1.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(tmpDir, "master-standalone"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "master-standalone", "example.io-test-stable-v1.json"), []byte(`{}`), 0o644)

	err := Generate(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "index.html"))
	html := string(data)
	if strings.Contains(html, "master-standalone") {
		t.Fatal("index should not list master-standalone directory")
	}
}
