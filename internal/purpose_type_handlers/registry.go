package purpose_type_handlers

import "fmt"

// PurposeHandlerRegistry holds all registered type handlers
type PurposeHandlerRegistry struct {
	handlers map[string]PurposeTypeHandler
}

var (
	// defaultRegistry is the global registry singleton
	defaultRegistry *PurposeHandlerRegistry
)

// init registers all built-in handlers at package init time
func init() {
	defaultRegistry = NewPurposeHandlerRegistry()

	// Register built-in handlers
	_ = defaultRegistry.Register(&StringPurposeTypeHandler{})
	_ = defaultRegistry.Register(&JsonSchemaPurposeTypeHandler{})
	_ = defaultRegistry.Register(&AttributePurposeTypeHandler{})
}

// NewPurposeHandlerRegistry creates a new registry instance
func NewPurposeHandlerRegistry() *PurposeHandlerRegistry {
	return &PurposeHandlerRegistry{
		handlers: make(map[string]PurposeTypeHandler),
	}
}

// Register adds a handler to the registry
// Returns error if a handler for this type is already registered
func (r *PurposeHandlerRegistry) Register(handler PurposeTypeHandler) error {
	typeStr := handler.GetType()
	if _, exists := r.handlers[typeStr]; exists {
		return fmt.Errorf("handler for type %q already registered", typeStr)
	}
	r.handlers[typeStr] = handler
	return nil
}

// Get retrieves a handler by type string
// Returns error if no handler is registered for the type
func (r *PurposeHandlerRegistry) Get(typeStr string) (PurposeTypeHandler, error) {
	handler, exists := r.handlers[typeStr]
	if !exists {
		return nil, fmt.Errorf("no handler registered for purpose type %q", typeStr)
	}
	return handler, nil
}

// GetAllTypes returns a list of all registered purpose types
func (r *PurposeHandlerRegistry) GetAllTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for typeStr := range r.handlers {
		types = append(types, typeStr)
	}
	return types
}

// Global helper functions

// GetHandler retrieves a handler from the default registry by type
func GetHandler(typeStr string) (PurposeTypeHandler, error) {
	return defaultRegistry.Get(typeStr)
}

// GetAllHandlerTypes returns list of all registered types in default registry
func GetAllHandlerTypes() []string {
	return defaultRegistry.GetAllTypes()
}

// GetDefaultRegistry returns the global registry singleton
func GetDefaultRegistry() *PurposeHandlerRegistry {
	return defaultRegistry
}
