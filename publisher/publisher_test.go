package publisher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublish_FullFlow(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/projects/test-project"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"name": "test-project"}})
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/upload-token"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"jwt": "fake-jwt-token"}})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/check-missing"):
			var body struct {
				Hashes []string `json:"hashes"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": body.Hashes})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/upload"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": nil})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/upsert-hashes"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": nil})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/deployments"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"id": "deploy-123", "url": "https://test-project.pages.dev"}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, "example.io"), 0o755)
	_ = os.WriteFile(filepath.Join(tmpDir, "example.io", "test_v1.json"), []byte(`{"type":"object"}`), 0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(`<html></html>`), 0o644)

	p := &Publisher{
		APIToken: "fake-token", AccountID: "fake-account", ProjectName: "test-project",
		BaseURL: server.URL + "/client/v4", AssetsURL: server.URL + "/client/v4",
	}
	err := p.Publish(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCalls := []string{"GET", "upload-token", "check-missing", "upload", "upsert-hashes", "deployments"}
	callStr := fmt.Sprintf("%v", calls)
	for _, expected := range expectedCalls {
		if !strings.Contains(callStr, expected) {
			t.Errorf("expected call containing %q, calls were: %v", expected, calls)
		}
	}
}

func TestPublish_CreatesProjectIfMissing(t *testing.T) {
	projectCreated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/projects/new-project"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "errors": []interface{}{map[string]interface{}{"code": 8000007}}})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/projects") && !strings.Contains(r.URL.Path, "deployments"):
			projectCreated = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"name": "new-project"}})
		case strings.HasSuffix(r.URL.Path, "/upload-token"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"jwt": "fake-jwt"}})
		case strings.HasSuffix(r.URL.Path, "/check-missing"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": []string{}})
		case strings.HasSuffix(r.URL.Path, "/upsert-hashes"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": nil})
		case strings.HasSuffix(r.URL.Path, "/deployments"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{"id": "d1", "url": "https://new-project.pages.dev"}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": nil})
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, "example.io"), 0o755)
	_ = os.WriteFile(filepath.Join(tmpDir, "example.io", "test_v1.json"), []byte(`{}`), 0o644)

	p := &Publisher{
		APIToken: "fake-token", AccountID: "fake-account", ProjectName: "new-project",
		BaseURL: server.URL + "/client/v4", AssetsURL: server.URL + "/client/v4",
	}
	err := p.Publish(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !projectCreated {
		t.Fatal("expected project to be created")
	}
}
