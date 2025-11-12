package purpose_type_handlers

// JsonSchemaPurposeTypeHandler handles "json-schema" type consent purposes
// JSON schema type requires validationSchema attribute to be present and valid JSON
type JsonSchemaPurposeTypeHandler struct{}

// GetType returns the type identifier
func (h *JsonSchemaPurposeTypeHandler) GetType() string {
	return "json-schema"
}

// ValidateAttributes validates attributes for json-schema type
// Mandatory: validationSchema must be present and valid JSON
func (h *JsonSchemaPurposeTypeHandler) ValidateAttributes(attributes map[string]string) []ValidationError {
	var errors []ValidationError

	// validationSchema is MANDATORY
	schema, exists := attributes["validationSchema"]
	if !exists || schema == "" {
		errors = append(errors, ValidationError{
			Field:   "validationSchema",
			Message: "validationSchema is required for json-schema type",
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

// ProcessAttributes processes attributes for json-schema type
// Could normalize JSON, add defaults, etc.
func (h *JsonSchemaPurposeTypeHandler) ProcessAttributes(attributes map[string]string) map[string]string {
	// Return as-is, basic processing
	return attributes
}

// GetAttributeSpec returns the attribute specification for json-schema type
func (h *JsonSchemaPurposeTypeHandler) GetAttributeSpec() []PurposeAttributeSpec {
	return []PurposeAttributeSpec{
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
