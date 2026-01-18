package validators

// AttributeElementTypeHandler handles "resource-field-type" consent elements
// Resource field type requires resourcePath and jsonPath to be present
type AttributeElementTypeHandler struct{}

// GetType returns the type identifier
func (h *AttributeElementTypeHandler) GetType() string {
	return "resource-field-type"
}

// ValidateProperties validates properties for resource-field-type
// Mandatory: resourcePath and jsonPath must be present
func (h *AttributeElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	var errors []ValidationError

	// resourcePath is MANDATORY
	if path, exists := properties["resourcePath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "resourcePath",
			Message: "resourcePath is required for resource-field-type",
		})
	}

	// jsonPath is MANDATORY
	if path, exists := properties["jsonPath"]; !exists || path == "" {
		errors = append(errors, ValidationError{
			Field:   "jsonPath",
			Message: "jsonPath is required for resource-field-type",
		})
	}

	return errors
}

// ProcessProperties processes properties for resource-field-type
// Basic processing, could add defaults or validation
func (h *AttributeElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is
	return properties
}

// GetPropertySpec returns the property specification for resource-field-type
func (h *AttributeElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
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
