package renderer

import (
	"fmt"
	"sort"
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

// PropertyEntry is a name+node pair for sorted iteration.
type PropertyEntry struct {
	Name string
	Node *SchemaNode
}

// IsRequired returns true if the given property name is in this node's required list.
func (n *SchemaNode) IsRequired(name string) bool {
	for _, r := range n.Required {
		if r == name {
			return true
		}
	}
	return false
}

// HasChildren returns true if this node has nested properties to render.
func (n *SchemaNode) HasChildren() bool {
	if len(n.Properties) > 0 {
		return true
	}
	if n.Items != nil && len(n.Items.Properties) > 0 {
		return true
	}
	return false
}

// SortedProperties returns properties sorted alphabetically by name.
func (n *SchemaNode) SortedProperties() []PropertyEntry {
	entries := make([]PropertyEntry, 0, len(n.Properties))
	for name, node := range n.Properties {
		entries = append(entries, PropertyEntry{Name: name, Node: node})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}

// Constraints returns human-readable constraint strings for display.
func (n *SchemaNode) Constraints() []string {
	var cs []string
	if len(n.Enum) > 0 {
		vals := make([]string, len(n.Enum))
		for i, v := range n.Enum {
			vals[i] = fmt.Sprintf("%v", v)
		}
		cs = append(cs, "enum: "+strings.Join(vals, ", "))
	}
	if n.Pattern != "" {
		cs = append(cs, "pattern: "+n.Pattern)
	}
	if n.Format != "" {
		cs = append(cs, "format: "+n.Format)
	}
	if n.MinLength != nil {
		cs = append(cs, fmt.Sprintf("minLength: %d", *n.MinLength))
	}
	if n.MaxLength != nil {
		cs = append(cs, fmt.Sprintf("maxLength: %d", *n.MaxLength))
	}
	if n.MinItems != nil {
		cs = append(cs, fmt.Sprintf("minItems: %d", *n.MinItems))
	}
	if n.MaxItems != nil {
		cs = append(cs, fmt.Sprintf("maxItems: %d", *n.MaxItems))
	}
	if n.Minimum != nil {
		cs = append(cs, fmt.Sprintf("minimum: %g", *n.Minimum))
	}
	if n.Maximum != nil {
		cs = append(cs, fmt.Sprintf("maximum: %g", *n.Maximum))
	}
	return cs
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
