package validators

import "encoding/json"

// ValidationError represents a single validation error for a property
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PurposePropertySpec defines metadata about a property for a purpose type
type PurposePropertySpec struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Type        string `json:"type"` // "string", "json", etc.
	Description string `json:"description"`
	Example     string `json:"example"`
}

// PurposeTypeHandler defines behavior for a specific consent purpose type
type PurposeTypeHandler interface {
	// GetType returns the type string this handler manages (e.g., "string-type", "json-payload-type", "resource-field-type")
	GetType() string

	// ValidateProperties checks if required properties are present and valid
	// Returns ValidationErrors if validation fails, empty slice if valid
	ValidateProperties(properties map[string]string) []ValidationError

	// ProcessProperties transforms/normalizes properties before storage
	// Useful for sanitization, defaults, or derived values
	ProcessProperties(properties map[string]string) map[string]string

	// GetPropertySpec returns the schema/spec for this handler's properties
	// Useful for documentation and dynamic UI generation
	GetPropertySpec() []PurposePropertySpec
}

// Helper function to validate JSON string
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
