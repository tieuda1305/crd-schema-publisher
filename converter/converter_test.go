package converter

import (
	"testing"
)

func TestAdditionalProperties_AddsToObjectWithProperties(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}
	result := AdditionalProperties(schema, false)
	val, ok := result["additionalProperties"]
	if !ok {
		t.Fatal("expected additionalProperties to be set")
	}
	if val != false {
		t.Fatalf("expected additionalProperties=false, got %v", val)
	}
}

func TestAdditionalProperties_DoesNotOverwriteExisting(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"additionalProperties": true,
	}
	result := AdditionalProperties(schema, false)
	if result["additionalProperties"] != true {
		t.Fatal("should not overwrite existing additionalProperties")
	}
}

func TestAdditionalProperties_SkipsObjectWithoutProperties(t *testing.T) {
	schema := map[string]interface{}{
		"type":        "object",
		"description": "no properties field",
	}
	result := AdditionalProperties(schema, false)
	if _, ok := result["additionalProperties"]; ok {
		t.Fatal("should not add additionalProperties when no properties field exists")
	}
}

func TestAdditionalProperties_RecursesIntoNestedObjects(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"spec": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"replicas": map[string]interface{}{"type": "integer"},
				},
			},
		},
	}
	result := AdditionalProperties(schema, false)
	spec := result["properties"].(map[string]interface{})["spec"].(map[string]interface{})
	if spec["additionalProperties"] != false {
		t.Fatal("nested object should also get additionalProperties=false")
	}
}

func TestAdditionalProperties_SkipsRootWhenFlagged(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}
	result := AdditionalProperties(schema, true)
	if _, ok := result["additionalProperties"]; ok {
		t.Fatal("should skip root when skip=true")
	}
}
