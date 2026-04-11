package renderer

import (
	"testing"
)

func TestDisplayType_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		typVal   interface{}
		items    *SchemaNode
		oneOf    []*SchemaNode
		expected string
	}{
		{"string", "string", nil, nil, "string"},
		{"integer", "integer", nil, nil, "integer"},
		{"boolean", "boolean", nil, nil, "boolean"},
		{"number", "number", nil, nil, "number"},
		{"object", "object", nil, nil, "object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &SchemaNode{Type: tt.typVal, Items: tt.items, OneOf: tt.oneOf}
			if got := n.DisplayType(); got != tt.expected {
				t.Errorf("DisplayType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDisplayType_NullableTypes(t *testing.T) {
	tests := []struct {
		name     string
		typVal   interface{}
		expected string
	}{
		{"nullable string", []interface{}{"string", "null"}, "string"},
		{"nullable integer", []interface{}{"integer", "null"}, "integer"},
		{"nullable object", []interface{}{"object", "null"}, "object"},
		{"null first", []interface{}{"null", "boolean"}, "boolean"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &SchemaNode{Type: tt.typVal}
			if got := n.DisplayType(); got != tt.expected {
				t.Errorf("DisplayType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDisplayType_Arrays(t *testing.T) {
	tests := []struct {
		name     string
		items    *SchemaNode
		expected string
	}{
		{"array of strings", &SchemaNode{Type: "string"}, "[]string"},
		{"array of objects", &SchemaNode{Type: "object"}, "[]object"},
		{"array of integers", &SchemaNode{Type: "integer"}, "[]integer"},
		{"array no items", nil, "[]object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &SchemaNode{Type: "array", Items: tt.items}
			if got := n.DisplayType(); got != tt.expected {
				t.Errorf("DisplayType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDisplayType_IntOrString(t *testing.T) {
	n := &SchemaNode{
		OneOf: []*SchemaNode{
			{Type: "string"},
			{Type: "integer"},
		},
	}
	if got := n.DisplayType(); got != "string | integer" {
		t.Errorf("DisplayType() = %q, want %q", got, "string | integer")
	}
}

func TestDisplayType_NoType(t *testing.T) {
	n := &SchemaNode{}
	if got := n.DisplayType(); got != "object" {
		t.Errorf("DisplayType() = %q, want %q", got, "object")
	}
}

func TestIsRequired(t *testing.T) {
	parent := &SchemaNode{
		Required: []string{"name", "spec"},
	}
	if !parent.IsRequired("name") {
		t.Error("name should be required")
	}
	if !parent.IsRequired("spec") {
		t.Error("spec should be required")
	}
	if parent.IsRequired("status") {
		t.Error("status should not be required")
	}
}

func TestHasChildren(t *testing.T) {
	tests := []struct {
		name     string
		node     *SchemaNode
		expected bool
	}{
		{"object with properties", &SchemaNode{Type: "object", Properties: map[string]*SchemaNode{"a": {}}}, true},
		{"object no properties", &SchemaNode{Type: "object"}, false},
		{"array with object items", &SchemaNode{Type: "array", Items: &SchemaNode{Type: "object", Properties: map[string]*SchemaNode{"a": {}}}}, true},
		{"array with string items", &SchemaNode{Type: "array", Items: &SchemaNode{Type: "string"}}, false},
		{"string", &SchemaNode{Type: "string"}, false},
		{"nullable object with props", &SchemaNode{Type: []interface{}{"object", "null"}, Properties: map[string]*SchemaNode{"a": {}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.HasChildren(); got != tt.expected {
				t.Errorf("HasChildren() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSortedProperties(t *testing.T) {
	node := &SchemaNode{
		Properties: map[string]*SchemaNode{
			"zeta":  {Type: "string"},
			"alpha": {Type: "string"},
			"mu":    {Type: "string"},
		},
	}
	sorted := node.SortedProperties()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(sorted))
	}
	expected := []string{"alpha", "mu", "zeta"}
	for i, p := range sorted {
		if p.Name != expected[i] {
			t.Errorf("property %d: got %q, want %q", i, p.Name, expected[i])
		}
	}
}

func TestConstraints(t *testing.T) {
	minLen := int64(1)
	maxLen := int64(255)
	min := 0.0
	max := 100.0

	tests := []struct {
		name     string
		node     *SchemaNode
		expected []string
	}{
		{"enum", &SchemaNode{Enum: []interface{}{"TCP", "UDP"}}, []string{"enum: TCP, UDP"}},
		{"pattern", &SchemaNode{Pattern: "^[a-z]+$"}, []string{"pattern: ^[a-z]+$"}},
		{"format", &SchemaNode{Format: "date-time"}, []string{"format: date-time"}},
		{"min/max length", &SchemaNode{MinLength: &minLen, MaxLength: &maxLen}, []string{"minLength: 1", "maxLength: 255"}},
		{"min/max value", &SchemaNode{Minimum: &min, Maximum: &max}, []string{"minimum: 0", "maximum: 100"}},
		{"no constraints", &SchemaNode{Type: "string"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.Constraints()
			if len(got) != len(tt.expected) {
				t.Fatalf("Constraints() returned %d items, want %d: %v", len(got), len(tt.expected), got)
			}
			for i, c := range got {
				if c != tt.expected[i] {
					t.Errorf("constraint %d: got %q, want %q", i, c, tt.expected[i])
				}
			}
		})
	}
}
