package constants

const (
	// HTTP Headers
	AuthorizationHeaderName = "Authorization"
	ContentTypeHeaderName   = "Content-Type"
	CorrelationIDHeaderName = "X-Correlation-ID"
	OrgIDHeaderName         = "org-id"
	TPPClientIDHeaderName   = "client-id"

	// Content Types
	ContentTypeJSON = "application/json"

	// Pagination
	DefaultPageSize = 30
	MaxPageSize     = 100

	// Token Types
	TokenTypeBearer = "Bearer"

	// API Base Path
	APIBasePath = "/api/v1"

	// Aliases for convenience
	HeaderContentType = ContentTypeHeaderName
	HeaderOrgID       = OrgIDHeaderName
	HeaderTPPClientID = TPPClientIDHeaderName
	ContentTypeJSON2  = ContentTypeJSON
)
