package validators

// StringElementTypeHandler handles "string-type" consent elements
// String type has no mandatory properties - all properties are optional
type StringElementTypeHandler struct{}

// GetType returns the type identifier
func (h *StringElementTypeHandler) GetType() string {
	return "string-type"
}

// ValidateProperties validates properties for string type
// String type has no mandatory properties, so validation always passes
func (h *StringElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	// String type: no mandatory properties
	// All properties are optional
	return nil
}

// ProcessProperties processes properties for string type
// No special processing needed for string type
func (h *StringElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is, no transformation needed
	return properties
}

// GetPropertySpec returns the property specification for string type
func (h *StringElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
		{
			Name:        "validationSchema",
			Required:    false,
			Type:        "json",
			Description: "JSON schema for validation",
			Example:     `{"type":"string","minLength":1}`,
		},
		{
			Name:        "resourcePath",
			Required:    false,
			Type:        "string",
			Description: "Resource path for this element",
			Example:     "/accounts",
		},
		{
			Name:        "jsonPath",
			Required:    false,
			Type:        "string",
			Description: "JSON path for data extraction",
			Example:     "Data.amount",
		},
	}
}
