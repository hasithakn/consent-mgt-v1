package purpose_type_handlers

// AttributePurposeTypeHandler handles "attribute" type consent purposes
// Attribute type requires resourcePath and jsonPath to be present
type AttributePurposeTypeHandler struct{}

// GetType returns the type identifier
func (h *AttributePurposeTypeHandler) GetType() string {
	return "attribute"
}

// ValidateAttributes validates attributes for attribute type
// Mandatory: resourcePath and jsonPath must be present
func (h *AttributePurposeTypeHandler) ValidateAttributes(attributes map[string]string) []ValidationError {
	var errors []ValidationError

	// resourcePath is MANDATORY
	if path, exists := attributes["resourcePath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "resourcePath",
			Message: "resourcePath is required for attribute type",
		})
	}

	// jsonPath is MANDATORY
	if path, exists := attributes["jsonPath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "jsonPath",
			Message: "jsonPath is required for attribute type",
		})
	}

	return errors
}

// ProcessAttributes processes attributes for attribute type
// Basic processing, could add defaults or validation
func (h *AttributePurposeTypeHandler) ProcessAttributes(attributes map[string]string) map[string]string {
	// Return as-is
	return attributes
}

// GetAttributeSpec returns the attribute specification for attribute type
func (h *AttributePurposeTypeHandler) GetAttributeSpec() []PurposeAttributeSpec {
	return []PurposeAttributeSpec{
		{
			Name:        "resourcePath",
			Required:    true,
			Type:        "string",
			Description: "Resource path (required)",
			Example:     "/accounts",
		},
		{
			Name:        "jsonPath",
			Required:    true,
			Type:        "string",
			Description: "JSON path for extraction (required)",
			Example:     "Data.amount",
		},
		{
			Name:        "validationSchema",
			Required:    false,
			Type:        "json",
			Description: "Optional validation schema",
			Example:     `{"type":"number"}`,
		},
	}
}
