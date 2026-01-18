package validators

import "fmt"

// ElementTypeHandlerRegistry holds all registered type handlers
type ElementTypeHandlerRegistry struct {
	handlers map[string]ElementTypeHandler
}

var (
	// defaultRegistry is the global registry singleton
	defaultRegistry *ElementTypeHandlerRegistry
)

// init registers all built-in handlers at package init time
func init() {
	defaultRegistry = NewElementTypeHandlerRegistry()

	// Register built-in handlers
	_ = defaultRegistry.Register(&StringElementTypeHandler{})
	_ = defaultRegistry.Register(&JsonSchemaElementTypeHandler{})
	_ = defaultRegistry.Register(&AttributeElementTypeHandler{})
}

// NewElementTypeHandlerRegistry creates a new registry instance
func NewElementTypeHandlerRegistry() *ElementTypeHandlerRegistry {
	return &ElementTypeHandlerRegistry{
		handlers: make(map[string]ElementTypeHandler),
	}
}

// Register adds a handler to the registry
// Returns error if a handler for this type is already registered
func (r *ElementTypeHandlerRegistry) Register(handler ElementTypeHandler) error {
	typeStr := handler.GetType()
	if _, exists := r.handlers[typeStr]; exists {
		return fmt.Errorf("handler for type %q already registered", typeStr)
	}
	r.handlers[typeStr] = handler
	return nil
}

// Get retrieves a handler by type string
// Returns error if no handler is registered for the type
func (r *ElementTypeHandlerRegistry) Get(typeStr string) (ElementTypeHandler, error) {
	handler, exists := r.handlers[typeStr]
	if !exists {
		return nil, fmt.Errorf("no handler registered for the element type %q", typeStr)
	}
	return handler, nil
}

// GetAllTypes returns a list of all registered element types
func (r *ElementTypeHandlerRegistry) GetAllTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for typeStr := range r.handlers {
		types = append(types, typeStr)
	}
	return types
}

// Global helper functions

// GetHandler retrieves a handler from the default registry by type
func GetHandler(typeStr string) (ElementTypeHandler, error) {
	return defaultRegistry.Get(typeStr)
}

// GetAllHandlerTypes returns list of all registered types in default registry
func GetAllHandlerTypes() []string {
	return defaultRegistry.GetAllTypes()
}

// GetDefaultRegistry returns the global registry singleton
func GetDefaultRegistry() *ElementTypeHandlerRegistry {
	return defaultRegistry
}
