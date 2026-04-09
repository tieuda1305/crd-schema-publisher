package converter

import "os"

// AdditionalProperties walks the schema tree and adds "additionalProperties": false
// to any object that has "properties" but lacks "additionalProperties".
// When skip is true, the root object is not modified (only its children are).
func AdditionalProperties(data map[string]interface{}, skip bool) map[string]interface{} {
	if _, hasProps := data["properties"]; hasProps && !skip {
		if _, hasAP := data["additionalProperties"]; !hasAP {
			data["additionalProperties"] = false
		}
	}
	for _, v := range data {
		if nested, ok := v.(map[string]interface{}); ok {
			AdditionalProperties(nested, false)
		}
	}
	return data
}

// ReplaceIntOrString finds {"format": "int-or-string"} and replaces with
// {"oneOf": [{"type": "string"}, {"type": "integer"}]}.
func ReplaceIntOrString(data interface{}) interface{} {
	switch d := data.(type) {
	case map[string]interface{}:
		if fmt, ok := d["format"]; ok && fmt == "int-or-string" {
			delete(d, "format")
			d["oneOf"] = []interface{}{
				map[string]interface{}{"type": "string"},
				map[string]interface{}{"type": "integer"},
			}
			return d
		}
		for k, v := range d {
			d[k] = ReplaceIntOrString(v)
		}
		return d
	case []interface{}:
		for i, v := range d {
			d[i] = ReplaceIntOrString(v)
		}
		return d
	default:
		return data
	}
}

// AllowNullOptionalFields converts non-required fields with "type": "X" to
// "type": ["X", "null"]. This makes optional fields nullable in the JSON schema.
// The requiredParent parameter is the nearest ancestor that may contain a "required" list.
func AllowNullOptionalFields(data interface{}, parent, requiredParent map[string]interface{}, key string) interface{} {
	switch d := data.(type) {
	case map[string]interface{}:
		// When this map has "properties", recurse into each property
		// passing this map as the requiredParent (since "required" is a sibling of "properties").
		if props, ok := d["properties"].(map[string]interface{}); ok {
			for pk, pv := range props {
				props[pk] = AllowNullOptionalFields(pv, props, d, pk)
			}
		}
		// Recurse into all other values (non-properties children)
		for k, v := range d {
			if k == "properties" {
				continue // already handled above
			}
			d[k] = AllowNullOptionalFields(v, d, requiredParent, k)
		}
		return d
	case []interface{}:
		for i, v := range d {
			d[i] = AllowNullOptionalFields(v, nil, requiredParent, key)
		}
		return d
	case string:
		if key == "type" && d != "null" {
			if requiredParent != nil {
				if _, hasRequired := requiredParent["required"]; hasRequired {
					return d
				}
			}
			return []interface{}{d, "null"}
		}
		return d
	default:
		return data
	}
}

// Convert applies all three transforms to a CRD OpenAPI schema in the correct order.
func Convert(schema map[string]interface{}) map[string]interface{} {
	skipRoot := os.Getenv("DENY_ROOT_ADDITIONAL_PROPERTIES") == ""
	schema = AdditionalProperties(schema, skipRoot)
	ReplaceIntOrString(schema)
	AllowNullOptionalFields(schema, nil, nil, "")
	return schema
}
