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
