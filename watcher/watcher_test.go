package watcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sholdee/crd-schema-publisher/publisher"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- helpers ---

type fakeLister struct {
	crds []apiextensionsv1.CustomResourceDefinition
	err  error
}

func (f *fakeLister) List(_ context.Context, _ metav1.ListOptions) (*apiextensionsv1.CustomResourceDefinitionList, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &apiextensionsv1.CustomResourceDefinitionList{Items: f.crds}, nil
}

func testCRD() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "tests.example.io"},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "Test"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name: "v1",
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {Type: "object"},
						},
					},
				},
			}},
		},
	}
}

// --- debounce tests (existing) ---

func TestDebounce_CoalescesRapidEvents(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	go debounceLoop(trigger, 100*time.Millisecond, func() error {
		count.Add(1)
		return nil
	}, done)

	// Send 5 events in rapid succession
	for range 5 {
		trigger <- struct{}{}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire
	time.Sleep(300 * time.Millisecond)
	close(done)

	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle, got %d", c)
	}
}

func TestDebounce_SkipsWhenPublishInProgress(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	go debounceLoop(trigger, 50*time.Millisecond, func() error {
		count.Add(1)
		// Simulate slow publish
		time.Sleep(300 * time.Millisecond)
		return nil
	}, done)

	// First event triggers publish
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires, publish starts (takes 300ms)

	// Second event during publish — debounce fires but publish in progress, skip
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires while first publish still running

	// Wait for everything to settle
	time.Sleep(500 * time.Millisecond)
	close(done)

	// Only 1 publish should have run (second was skipped)
	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle (second skipped), got %d", c)
	}
}

func TestDebounce_DrainsInFlightPublishOnShutdown(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	var loopDone sync.WaitGroup
	loopDone.Add(1)

	go func() {
		defer loopDone.Done()
		debounceLoop(trigger, 50*time.Millisecond, func() error {
			count.Add(1)
			// Simulate slow publish
			time.Sleep(500 * time.Millisecond)
			return nil
		}, done)
	}()

	// Trigger a publish
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires, publish starts

	// Shut down while publish is in progress
	close(done)

	// debounceLoop should block until publish completes, then return
	loopDone.Wait()

	// The in-flight publish must have completed (not been killed)
	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle to complete during drain, got %d", c)
	}
}

// --- publishCycle tests ---

func TestPublishCycle_HappyPath(t *testing.T) {
	t.Setenv("SKIP_RENDER", "true")
	dir := t.TempDir()

	cfg := Config{
		OutputDir: dir,
		CRDLister: &fakeLister{crds: []apiextensionsv1.CustomResourceDefinition{testCRD()}},
	}

	if err := publishCycle(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify schema files exist
	schemaPath := filepath.Join(dir, "example.io", "test_v1.json")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatalf("expected schema file at %s", schemaPath)
	}

	// Verify index.html exists
	indexPath := filepath.Join(dir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("expected index.html at %s", indexPath)
	}
}

func TestPublishCycle_ExtractError(t *testing.T) {
	t.Setenv("SKIP_RENDER", "true")
	dir := t.TempDir()

	cfg := Config{
		OutputDir: dir,
		CRDLister: &fakeLister{err: fmt.Errorf("API unavailable")},
	}

	err := publishCycle(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "listing CRDs") {
		t.Fatalf("expected error containing 'listing CRDs', got: %s", got)
	}

	// Verify no schema files written
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %d entries", len(entries))
	}
}

func TestPublishCycle_EmptyCRDs(t *testing.T) {
	t.Setenv("SKIP_RENDER", "true")
	dir := t.TempDir()

	cfg := Config{
		OutputDir: dir,
		CRDLister: &fakeLister{crds: []apiextensionsv1.CustomResourceDefinition{}},
	}

	if err := publishCycle(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("expected empty dir after empty CRD list, got %d entries", len(entries))
	}
}

func TestPublishCycle_UploadError(t *testing.T) {
	t.Setenv("SKIP_RENDER", "true")
	dir := t.TempDir()

	// Mock server that returns 500 for all requests
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"message":"server error"}]}`))
	}))
	defer srv.Close()

	cfg := Config{
		OutputDir: dir,
		CRDLister: &fakeLister{crds: []apiextensionsv1.CustomResourceDefinition{testCRD()}},
		Publisher: &publisher.Publisher{
			BaseURL:     srv.URL,
			AssetsURL:   srv.URL,
			APIToken:    "t",
			AccountID:   "a",
			ProjectName: "p",
			SleepFunc:   func(time.Duration) {},
		},
	}

	err := publishCycle(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "publishing") {
		t.Fatalf("expected error containing 'publishing', got: %s", got)
	}
}

// --- cleanDir tests ---

func TestCleanDir_RemovesContents(t *testing.T) {
	dir := t.TempDir()

	// Create some files and subdirs
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cleanDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dir should exist but be empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %d entries", len(entries))
	}
}

func TestCleanDir_CreatesIfNotExist(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent", "nested")

	if err := cleanDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected dir to be created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}
