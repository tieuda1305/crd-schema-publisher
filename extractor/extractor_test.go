package extractor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeCRD() apiextensionsv1.CustomResourceDefinition {
	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"spec": {
				Type: "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{
					"replicas": {Type: "integer"},
				},
			},
		},
	}
	return apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "tests.example.io"},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "Test"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name: "v1",
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: schema,
					},
				},
			},
		},
	}
}

func TestWriteSchemas_CreatesGroupDirAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	crds := []apiextensionsv1.CustomResourceDefinition{fakeCRD()}
	count, err := WriteSchemas(crds, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 schema, got %d", count)
	}
	schemaPath := filepath.Join(tmpDir, "example.io", "test_v1.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("schema file not found: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// After conversion, type may be "object" or ["object", "null"]
	switch v := schema["type"].(type) {
	case string:
		if v != "object" {
			t.Fatalf("expected type=object, got %v", v)
		}
	case []interface{}:
		found := false
		for _, item := range v {
			if item == "object" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected type to contain 'object', got %v", v)
		}
	default:
		t.Fatalf("unexpected type field: %T %v", schema["type"], schema["type"])
	}
}

func TestWriteSchemas_CreatesMasterStandalone(t *testing.T) {
	tmpDir := t.TempDir()
	crds := []apiextensionsv1.CustomResourceDefinition{fakeCRD()}
	_, _ = WriteSchemas(crds, tmpDir)
	standalonePath := filepath.Join(tmpDir, "master-standalone", "example.io-test-stable-v1.json")
	if _, err := os.Stat(standalonePath); os.IsNotExist(err) {
		t.Fatalf("master-standalone file not found: %s", standalonePath)
	}
}
