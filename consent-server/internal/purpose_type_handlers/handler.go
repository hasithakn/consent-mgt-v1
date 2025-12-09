package purpose_type_handlers

import "encoding/json"

// ValidationError represents a single validation error for an attribute
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PurposeAttributeSpec defines metadata about an attribute for a purpose type
type PurposeAttributeSpec struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Type        string `json:"type"` // "string", "json", etc.
	Description string `json:"description"`
	Example     string `json:"example"`
}

// PurposeTypeHandler defines behavior for a specific consent purpose type
type PurposeTypeHandler interface {
	// GetType returns the type string this handler manages (e.g., "string", "json-schema", "attribute")
	GetType() string

	// ValidateAttributes checks if required attributes are present and valid
	// Returns ValidationErrors if validation fails, empty slice if valid
	ValidateAttributes(attributes map[string]string) []ValidationError

	// ProcessAttributes transforms/normalizes attributes before storage
	// Useful for sanitization, defaults, or derived values
	ProcessAttributes(attributes map[string]string) map[string]string

	// GetAttributeSpec returns the schema/spec for this handler's attributes
	// Useful for documentation and dynamic UI generation
	GetAttributeSpec() []PurposeAttributeSpec
}

// Helper function to validate JSON string
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
