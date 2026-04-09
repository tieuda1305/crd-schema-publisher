package converter

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
