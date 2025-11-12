package purpose_type_handlers

// StringPurposeTypeHandler handles "string" type consent purposes
// String type has no mandatory attributes - all attributes are optional
type StringPurposeTypeHandler struct{}

// GetType returns the type identifier
func (h *StringPurposeTypeHandler) GetType() string {
	return "string"
}

// ValidateAttributes validates attributes for string type
// String type has no mandatory attributes, so validation always passes
func (h *StringPurposeTypeHandler) ValidateAttributes(attributes map[string]string) []ValidationError {
	// String type: no mandatory attributes
	// All attributes are optional
	return nil
}

// ProcessAttributes processes attributes for string type
// No special processing needed for string type
func (h *StringPurposeTypeHandler) ProcessAttributes(attributes map[string]string) map[string]string {
	// Return as-is, no transformation needed
	return attributes
}

// GetAttributeSpec returns the attribute specification for string type
func (h *StringPurposeTypeHandler) GetAttributeSpec() []PurposeAttributeSpec {
	return []PurposeAttributeSpec{
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
			Description: "Resource path for this purpose",
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
