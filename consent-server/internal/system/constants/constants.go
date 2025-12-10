package constants

const (
	AuthorizationHeaderName = "Authorization"
	ContentTypeHeaderName   = "Content-Type"
	CorrelationIDHeaderName = "X-Correlation-ID"
	OrgIDHeaderName         = "X-Organization-ID"
	ContentTypeJSON         = "application/json"
	DefaultPageSize         = 30
	MaxPageSize             = 100
	TokenTypeBearer         = "Bearer"

	// Aliases for convenience
	HeaderContentType = ContentTypeHeaderName
	HeaderOrgID       = OrgIDHeaderName
	ContentTypeJSON2  = ContentTypeJSON
)
