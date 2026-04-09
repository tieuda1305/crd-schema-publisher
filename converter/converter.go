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
