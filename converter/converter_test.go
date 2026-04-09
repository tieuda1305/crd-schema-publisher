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

func TestReplaceIntOrString_ReplacesFormat(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"port": map[string]interface{}{
				"format": "int-or-string",
			},
		},
	}
	result := ReplaceIntOrString(schema)
	port := result.(map[string]interface{})["properties"].(map[string]interface{})["port"].(map[string]interface{})
	oneOf, ok := port["oneOf"]
	if !ok {
		t.Fatal("expected oneOf to replace int-or-string format")
	}
	items := oneOf.([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 oneOf items, got %d", len(items))
	}
	if _, hasFormat := port["format"]; hasFormat {
		t.Fatal("format field should be removed")
	}
}

func TestReplaceIntOrString_LeavesOtherFormats(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
	}
	result := ReplaceIntOrString(schema)
	name := result.(map[string]interface{})["properties"].(map[string]interface{})["name"].(map[string]interface{})
	if name["format"] != "date-time" {
		t.Fatal("should preserve non-int-or-string format")
	}
}

func TestReplaceIntOrString_RecursesIntoArrayItems(t *testing.T) {
	schema := map[string]interface{}{
		"items": map[string]interface{}{
			"format": "int-or-string",
		},
	}
	result := ReplaceIntOrString(schema)
	items := result.(map[string]interface{})["items"].(map[string]interface{})
	if _, ok := items["oneOf"]; !ok {
		t.Fatal("should recurse into nested objects")
	}
}

func TestAllowNullOptionalFields_ConvertsNonRequiredType(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	result := AllowNullOptionalFields(schema, nil, nil, "")
	name := result.(map[string]interface{})["properties"].(map[string]interface{})["name"].(map[string]interface{})
	typeVal := name["type"]
	arr, ok := typeVal.([]interface{})
	if !ok {
		t.Fatalf("expected type to be array, got %T", typeVal)
	}
	if len(arr) != 2 || arr[0] != "string" || arr[1] != "null" {
		t.Fatalf("expected [string, null], got %v", arr)
	}
}

func TestAllowNullOptionalFields_SkipsRequiredFields(t *testing.T) {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	result := AllowNullOptionalFields(schema, nil, nil, "")
	name := result.(map[string]interface{})["properties"].(map[string]interface{})["name"].(map[string]interface{})
	if name["type"] != "string" {
		t.Fatal("required field type should remain a string, not array")
	}
}

func TestAllowNullOptionalFields_SkipsNullType(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"nothing": map[string]interface{}{
				"type": "null",
			},
		},
	}
	result := AllowNullOptionalFields(schema, nil, nil, "")
	nothing := result.(map[string]interface{})["properties"].(map[string]interface{})["nothing"].(map[string]interface{})
	if nothing["type"] != "null" {
		t.Fatal("null type should not be modified")
	}
}

func TestConvert_AppliesAllTransforms(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"port": map[string]interface{}{
				"format": "int-or-string",
			},
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	result := Convert(schema)
	port := result["properties"].(map[string]interface{})["port"].(map[string]interface{})
	if _, ok := port["oneOf"]; !ok {
		t.Fatal("intOrString not applied")
	}
	name := result["properties"].(map[string]interface{})["name"].(map[string]interface{})
	typeVal := name["type"]
	if _, ok := typeVal.([]interface{}); !ok {
		t.Fatal("nullOptional not applied")
	}
}
