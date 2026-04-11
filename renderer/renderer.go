package renderer

import (
	"strings"
)

// SchemaNode represents a JSON Schema node for rendering.
type SchemaNode struct {
	Type                 interface{}            `json:"type,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]*SchemaNode `json:"properties,omitempty"`
	Items                *SchemaNode            `json:"items,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	OneOf                []*SchemaNode          `json:"oneOf,omitempty"`
	AnyOf                []*SchemaNode          `json:"anyOf,omitempty"`
	AllOf                []*SchemaNode          `json:"allOf,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	MinLength            *int64                 `json:"minLength,omitempty"`
	MaxLength            *int64                 `json:"maxLength,omitempty"`
	MinItems             *int64                 `json:"minItems,omitempty"`
	MaxItems             *int64                 `json:"maxItems,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
}

// DisplayType returns a human-readable type string for the schema node.
func (n *SchemaNode) DisplayType() string {
	if len(n.OneOf) == 2 && n.Type == nil {
		types := make([]string, 0, 2)
		for _, o := range n.OneOf {
			types = append(types, o.DisplayType())
		}
		return strings.Join(types, " | ")
	}

	raw := n.resolveType()

	if raw == "array" {
		itemType := "object"
		if n.Items != nil {
			itemType = n.Items.resolveType()
		}
		return "[]" + itemType
	}

	if raw == "" {
		return "object"
	}

	return raw
}

// resolveType extracts the non-null type from the Type field.
func (n *SchemaNode) resolveType() string {
	switch t := n.Type.(type) {
	case string:
		return t
	case []interface{}:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}
