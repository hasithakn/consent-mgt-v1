package validators

// JsonSchemaElementTypeHandler handles "json-payload-type" consent elements
// JSON payload type requires validationSchema property to be present and valid JSON
type JsonSchemaElementTypeHandler struct{}

// GetType returns the type identifier
func (h *JsonSchemaElementTypeHandler) GetType() string {
	return "json-payload-type"
}

// ValidateProperties validates properties for json-payload-type
// Mandatory: validationSchema must be present and valid JSON
func (h *JsonSchemaElementTypeHandler) ValidateProperties(properties map[string]string) []ValidationError {
	var errors []ValidationError

	// validationSchema is MANDATORY
	schema, exists := properties["validationSchema"]
	if !exists || schema == "" {
		errors = append(errors, ValidationError{
			Field:   "validationSchema",
			Message: "validationSchema is required for json-payload-type",
		})
		return errors
	}

	// Validate that validationSchema is valid JSON
	if !isValidJSON(schema) {
		errors = append(errors, ValidationError{
			Field:   "validationSchema",
			Message: "validationSchema must be valid JSON",
		})
	}

	return errors
}

// ProcessProperties processes properties for json-payload-type
// Could normalize JSON, add defaults, etc.
func (h *JsonSchemaElementTypeHandler) ProcessProperties(properties map[string]string) map[string]string {
	// Return as-is, basic processing
	return properties
}

// GetPropertySpec returns the property specification for json-payload-type
func (h *JsonSchemaElementTypeHandler) GetPropertySpec() []ElementPropertySpec {
	return []ElementPropertySpec{
		{
			Name:        "validationSchema",
			Required:    true,
			Type:        "json",
			Description: "JSON schema for validation (required)",
			Example:     `{"type":"object","properties":{"name":{"type":"string"}}}`,
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
