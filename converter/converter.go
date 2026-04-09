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
// propertyName is the field's key in its parent "properties" map.
// requiredSet contains the names of required fields from the parent object.
func AllowNullOptionalFields(data interface{}, propertyName string, requiredSet map[string]bool) interface{} {
	switch d := data.(type) {
	case map[string]interface{}:
		// If this property object has a "type" and is not required, make it nullable.
		if propertyName != "" && !requiredSet[propertyName] {
			if t, ok := d["type"].(string); ok && t != "null" {
				d["type"] = []interface{}{t, "null"}
			}
		}

		// Build the required set for this object's own properties.
		var childRequired map[string]bool
		if reqList, ok := d["required"].([]interface{}); ok {
			childRequired = make(map[string]bool, len(reqList))
			for _, r := range reqList {
				if s, ok := r.(string); ok {
					childRequired[s] = true
				}
			}
		}

		// Recurse into properties with the required set from this object.
		if props, ok := d["properties"].(map[string]interface{}); ok {
			for pk, pv := range props {
				props[pk] = AllowNullOptionalFields(pv, pk, childRequired)
			}
		}

		// Recurse into non-property children (e.g., items, additionalProperties schema).
		for k, v := range d {
			if k == "properties" {
				continue
			}
			d[k] = AllowNullOptionalFields(v, "", nil)
		}
		return d
	case []interface{}:
		for i, v := range d {
			d[i] = AllowNullOptionalFields(v, "", nil)
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
	schema = ReplaceIntOrString(schema).(map[string]interface{})
	schema = AllowNullOptionalFields(schema, "", nil).(map[string]interface{})
	return schema
}
