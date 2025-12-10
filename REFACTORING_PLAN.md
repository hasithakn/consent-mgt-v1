# Consent Server Architecture Refactoring Plan

## Executive Summary

Transform consent-server from **layered architecture** to **resource-based package architecture** following Thunder project patterns with **http.ServeMux** and **standardized error handling**.

### Current State (Layered)
```
internal/
├── dao/              # Data access layer
├── handlers/         # HTTP handlers (Gin-based)
├── service/          # Business logic
├── models/           # Shared models
└── router/           # Centralized routing (Gin)
```

### Target State (Resource-Based + Thunder Patterns)
```
internal/
├── consent/
│   ├── store.go      # Data access: interface + implementation
│   ├── service.go    # Business logic: ConsentServiceInterface (exported) + impl
│   ├── handler.go    # HTTP handlers (http.Handler compatible)
│   ├── init.go       # Initialize() + registerRoutes()
│   └── model/        # Consent-specific models
├── consentpurpose/
│   ├── store.go
│   ├── service.go    # ConsentPurposeServiceInterface (exported)
│   ├── handler.go
│   ├── init.go
│   └── model/
├── authresource/
│   ├── store.go
│   ├── service.go    # AuthResourceServiceInterface (exported)
│   ├── handler.go
│   ├── init.go
│   └── model/
└── shared/           # Thunder-inspired common utilities
    ├── database/     # Multi-DB support, transactions, query builder
    ├── errors/       # ServiceError + ErrorResponse
    ├── middleware/   # CorrelationID, CORS
    ├── constants/    # HTTP headers, pagination, etc.
    └── utils/        # HTTP, security, validation helpers
```

### Key Architectural Changes

| Aspect | Current | Target |
|--------|---------|--------|
| **Router** | Gin framework | `http.ServeMux` (Go stdlib) |
| **Error Handling** | Standard `error` | `serviceerror.ServiceError` (2-tier) |
| **Middleware** | Gin middleware | Standard `http.Handler` middleware |
| **Route Registration** | Centralized `router.go` | Per-resource `init.go` |
| **Database** | sqlx with manual transactions | Thunder patterns: DBQuery + functional transactions |
| **Request Tracking** | Manual (partial) | Correlation ID middleware (automatic) |

---

## 1. Thunder Architecture Analysis

### 1.1 Main.go Flow (Entry Point)

Thunder's main.go demonstrates the complete wiring:

```go
func main() {
    logger := log.GetLogger()
    
    // 1. Load configurations
    cfg := initThunderConfigurations(logger, thunderHome)
    
    // 2. Initialize cache manager
    initCacheManager(logger)
    
    // 3. Create http.ServeMux (standard library router)
    mux := http.NewServeMux()
    
    // 4. Initialize JWT service
    jwtService := jwt.GetJWTService()
    jwtService.Init()
    
    // 5. Register ALL services (each calls resource.Initialize(mux, deps...))
    registerServices(mux, jwtService)
    
    // 6. Build middleware chain (reverse order)
    handler := mux                                      // Innermost: route handlers
    handler = log.AccessLogHandler(logger, handler)    // Access logging
    handler = middleware.CorrelationIDMiddleware(handler) // Outermost: correlation ID
    
    // 7. Create HTTP server
    server := &http.Server{
        Addr:    fmt.Sprintf("%s:%d", cfg.Server.Hostname, cfg.Server.Port),
        Handler: handler,
        ReadHeaderTimeout: 10 * time.Second,
        WriteTimeout:      10 * time.Second,
    }
    
    // 8. Start server (HTTP or HTTPS)
    server.ListenAndServe() // or TLS variant
    
    // 9. Graceful shutdown on signal
    <-sigChan
    gracefulShutdown(logger, server)
}
```

### 1.2 Service Registration Pattern

```go
// servicemanager.go
func registerServices(mux *http.ServeMux, jwtService jwt.JWTServiceInterface) {
    // Initialize services in dependency order
    ouService := ou.Initialize(mux)
    userSchemaService := userschema.Initialize(mux, ouService)
    userService := user.Initialize(mux, ouService, userSchemaService)
    groupService := group.Initialize(mux, ouService, userService)
    roleService := role.Initialize(mux, userService, groupService, ouService)
    
    // Initialize with multiple dependencies
    certService := cert.Initialize()
    flowMgtService, _ := flowmgt.Initialize(flowFactory, execRegistry)
    brandingService := branding.Initialize(mux)
    
    // Application service depends on multiple services
    applicationService := application.Initialize(
        mux,
        certService,
        flowMgtService,
        brandingService,
        userSchemaService,
    )
    
    // OAuth depends on application service
    oauth.Initialize(mux, applicationService, userService, jwtService, flowExecService)
}
```

**Key Observations:**
- ✅ Each `Initialize()` call registers routes AND returns service interface
- ✅ Dependencies passed explicitly (dependency injection)
- ✅ Services initialized in correct order (dependencies first)
- ✅ Single source of truth for all registrations

### 1.3 Key Patterns Identified

#### **Package-per-Resource Pattern**
- Each domain resource has its own package (e.g., `application/`)
- All related code colocated: store, service, handler, models
- Self-contained with clear boundaries

#### **Store Pattern (Data Access Layer)**
```go
// Interface defines contract
type applicationStoreInterface interface {
    CreateApplication(...) error
    GetApplicationByID(id string) (*model.Application, error)
    UpdateApplication(...) error
    DeleteApplication(id string) error
}

// Implementation
type applicationStore struct {
    deploymentID string
}

// Factory function (lowercase = unexported)
func newApplicationStore() applicationStoreInterface {
    return &applicationStore{...}
}
```

**Key Characteristics:**
- Interface-based design for testability
- Multiple implementations possible (database, cached, file-based)
- Factory functions are unexported (internal to package)
- No exported constructors - use Initialize() instead

#### **Service Pattern (Business Logic)**
```go
// Interface defines business operations
type ApplicationServiceInterface interface {
    CreateApplication(...) (*model.ApplicationDTO, *serviceerror.ServiceError)
    GetApplication(id string) (*model.Application, *serviceerror.ServiceError)
    UpdateApplication(...) (*model.ApplicationDTO, *serviceerror.ServiceError)
    DeleteApplication(id string) *serviceerror.ServiceError
}

// Implementation with dependencies
type applicationService struct {
    appStore          applicationStoreInterface
    certService       cert.CertificateServiceInterface
    flowMgtService    flowmgt.FlowMgtServiceInterface
    brandingService   branding.BrandingServiceInterface
}

// Factory
func newApplicationService(
    appStore applicationStoreInterface,
    certService cert.CertificateServiceInterface,
    ...,
) ApplicationServiceInterface {
    return &applicationService{...}
}
```

**Key Characteristics:**
- Interface exported (public contract)
- Implementation struct unexported (internal)
- Depends on store interface + other service interfaces
- Returns domain errors (not HTTP errors)

#### **Handler Pattern (HTTP Layer)**
```go
// Handler struct
type applicationHandler struct {
    service ApplicationServiceInterface
}

// Factory
func newApplicationHandler(service ApplicationServiceInterface) *applicationHandler {
    return &applicationHandler{service: service}
}

// Handler methods
func (h *applicationHandler) HandleApplicationPostRequest(w http.ResponseWriter, r *http.Request) {
    app, err := sysutils.DecodeJSONBody[model.ApplicationRequest](r)
    if err != nil {
        apierror.ErrorResponse(w, err)
        return
    }
    
    result, svcErr := h.service.CreateApplication(app)
    if svcErr != nil {
        apierror.ErrorResponse(w, svcErr)
        return
    }
    
    sysutils.JSONResponse(w, http.StatusCreated, result)
}
```

**Key Characteristics:**
- Unexported struct
- Depends only on service interface
- Uses utility functions for JSON encoding/decoding
- Translates service errors to HTTP responses

#### **Initialization Pattern**
```go
// Initialize wires up all dependencies and returns service interface
func Initialize(
    mux *http.ServeMux,
    certService cert.CertificateServiceInterface,
    flowMgtService flowmgt.FlowMgtServiceInterface,
    ...,
) ApplicationServiceInterface {
    // 1. Create store (choose implementation based on config)
    var appStore applicationStoreInterface
    if config.GetThunderRuntime().Config.ImmutableResources.Enabled {
        appStore = newFileBasedStore()
    } else {
        store := newApplicationStore()
        appStore = newCachedBackedApplicationStore(store) // Decorator pattern
    }
    
    // 2. Create service with dependencies
    appService := newApplicationService(appStore, certService, flowMgtService, ...)
    
    // 3. Create handler
    appHandler := newApplicationHandler(appService)
    
    // 4. Register routes
    registerRoutes(mux, appHandler)
    
    // 5. Return service interface (for other packages to depend on)
    return appService
}
```

**Key Characteristics:**
- **Single exported function** per resource package
- Centralizes dependency injection
- Returns service interface (for inter-package dependencies)
- Calls private registerRoutes()
- Handles configuration-based store selection

#### **Route Registration Pattern**
```go
func registerRoutes(mux *http.ServeMux, appHandler *applicationHandler) {
    opts1 := middleware.CORSOptions{
        AllowedMethods:   "GET, POST",
        AllowedHeaders:   "Content-Type, Authorization",
        AllowCredentials: true,
    }
    
    mux.HandleFunc(middleware.WithCORS("POST /applications",
        appHandler.HandleApplicationPostRequest, opts1))
    mux.HandleFunc(middleware.WithCORS("GET /applications",
        appHandler.HandleApplicationListRequest, opts1))
    
    // More routes...
}
```

**Key Characteristics:**
- Unexported function (called only by Initialize())
- Takes http.ServeMux (standard library router)
- Registers all routes for this resource
- Applies middleware per-route

### 1.2 Dependency Flow

```
main.go
  └─> Initialize() for each resource
        ├─> newStore()
        ├─> newService(store, otherServices...)
        ├─> newHandler(service)
        └─> registerRoutes(mux, handler)
```

### 1.3 Interface vs Implementation Visibility

| Component | Type | Visibility | Why |
|-----------|------|------------|-----|
| Service Interface | `ApplicationServiceInterface` | **Exported** | Public contract for other packages |
| Service Implementation | `applicationService` | **Unexported** | Internal implementation detail |
| Store Interface | `applicationStoreInterface` | **Unexported** | Only used within package |
| Store Implementation | `applicationStore` | **Unexported** | Internal implementation |
| Handler | `applicationHandler` | **Unexported** | Only used within package |
| Initialize() | `Initialize()` | **Exported** | Entry point for wiring |
| Factories | `newStore()`, `newService()`, `newHandler()` | **Unexported** | Internal construction |

---

## 2. Current Consent Server Architecture

### 2.1 Current Structure

```
consent-server/internal/
├── dao/
│   ├── consent_dao.go              (Create, Get, Update, Delete, Search)
│   ├── consent_purpose_dao.go      (Create, Get, Update, Delete, List)
│   ├── consent_purpose_attribute_dao.go
│   ├── consent_attribute_dao.go
│   ├── consent_file_dao.go
│   ├── auth_resource_dao.go
│   └── status_audit_dao.go
├── service/
│   ├── consent_service.go          (Business logic)
│   ├── consent_purpose_service.go
│   └── auth_resource_service.go
├── handlers/
│   ├── consent_handler.go          (HTTP handlers)
│   ├── consent_purpose_handler.go
│   └── auth_resource_handler.go
├── models/                          (Shared across all)
│   ├── consent.go
│   ├── consent_purpose.go
│   ├── consent_attribute.go
│   ├── auth_resource.go
│   └── common.go
└── router/
    └── router.go                    (Centralized route registration)
```

### 2.2 Current Dependency Chain

```
main.go
  ├─> Creates DAOs (consentDAO, purposeDAO, authResourceDAO, ...)
  ├─> Creates Services (consentService(consentDAO, statusAuditDAO, ...))
  ├─> Creates ExtensionClient
  └─> Calls router.SetupRouter(services..., extensionClient)
        ├─> Creates Handlers (consentHandler, purposeHandler, ...)
        └─> Registers all routes on gin.Engine
```

### 2.3 Issues with Current Architecture

1. **Tight Coupling**: DAOs created in main.go, passed to services, then router creates handlers
2. **Centralized Routing**: All routes in one file (router.go) - hard to maintain
3. **Shared Models**: All models in one package - changes affect everything
4. **No Interfaces**: DAOs are concrete structs, not interfaces (hard to test/mock)
5. **Mixed Concerns**: Service layer mixes business logic with data access details
6. **Framework Lock-in**: Gin framework used throughout (harder to switch)

---

## 3. Refactoring Strategy

### 3.1 Resource Identification

Based on current code, we have **3 main resources**:

1. **Consent** (consent, consent_attribute, consent_file, status_audit)
2. **ConsentPurpose** (consent_purpose, consent_purpose_attribute)
3. **AuthResource** (auth_resource)

### 3.2 Migration Order (Least Dependencies First)

#### **Phase 1: AuthResource** (Simplest - no dependencies)
- Standalone resource
- No dependencies on other domain resources
- Good candidate for testing the pattern

#### **Phase 2: ConsentPurpose** (Medium complexity)
- Depends on database only
- Used by Consent resource
- Intermediate complexity

#### **Phase 3: Consent** (Most complex)
- Depends on ConsentPurpose, AuthResource
- Has extension client integration
- Most routes and logic

### 3.3 Incremental Migration Steps

Each resource follows this pattern:

```
Step 1: Create new package structure
Step 2: Move models to resource/model/
Step 3: Create store.go (interface + implementation) from DAO
Step 4: Move service.go to resource package
Step 5: Move handler.go to resource package
Step 6: Create init.go with Initialize() and registerRoutes()
Step 7: Update main.go to call Initialize()
Step 8: Remove old code from dao/, service/, handlers/
Step 9: Test thoroughly
```

---

## 4. Detailed Refactoring Plan by Resource

### 4.1 Phase 1: AuthResource

#### **Before (Current):**
```
internal/
├── dao/
│   └── auth_resource_dao.go
├── service/
│   └── auth_resource_service.go
├── handlers/
│   └── auth_resource_handler.go
└── models/
    └── auth_resource.go
```

#### **After (Target):**
```
internal/
└── authresource/
    ├── store.go         (interface + implementation from auth_resource_dao.go)
    ├── service.go       (interface + implementation from auth_resource_service.go)
    ├── handler.go       (unexported struct from auth_resource_handler.go)
    ├── init.go          (Initialize() and registerRoutes())
    └── model/
        └── auth_resource.go
```

#### **Implementation Details:**

**store.go:**
```go
package authresource

import "github.com/wso2/consent-management-api/internal/system/database"

// authResourceStoreInterface defines data access operations (unexported)
type authResourceStoreInterface interface {
    Create(resource *model.AuthResource) error
    Get(id string, orgID string) (*model.AuthResource, error)
    List(orgID string, limit, offset int) ([]*model.AuthResource, error)
    Update(resource *model.AuthResource) error
    Delete(id string, orgID string) error
}

// authResourceStore implements the store interface
type authResourceStore struct {
    db *database.Database
}

// newAuthResourceStore creates a new store instance (unexported)
func newAuthResourceStore(db *database.Database) authResourceStoreInterface {
    return &authResourceStore{db: db}
}

// Implementation methods...
func (s *authResourceStore) Create(resource *model.AuthResource) error {
    // Move logic from auth_resource_dao.go
}
```

**service.go:**
```go
package authresource

// AuthResourceServiceInterface defines business operations (EXPORTED)
type AuthResourceServiceInterface interface {
    CreateAuthResource(resource *model.AuthResource) (*model.AuthResource, error)
    GetAuthResource(id, orgID string) (*model.AuthResource, error)
    ListAuthResources(orgID string, limit, offset int) ([]*model.AuthResource, error)
    UpdateAuthResource(resource *model.AuthResource) error
    DeleteAuthResource(id, orgID string) error
}

// authResourceService implements the service
type authResourceService struct {
    store authResourceStoreInterface
}

// newAuthResourceService creates service (unexported)
func newAuthResourceService(store authResourceStoreInterface) AuthResourceServiceInterface {
    return &authResourceService{store: store}
}

// Implementation methods...
func (s *authResourceService) CreateAuthResource(resource *model.AuthResource) (*model.AuthResource, error) {
    // Move logic from auth_resource_service.go
    return s.store.Create(resource)
}
```

**handler.go:**
```go
package authresource

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// authResourceHandler handles HTTP requests (unexported)
type authResourceHandler struct {
    service AuthResourceServiceInterface
}

// newAuthResourceHandler creates handler (unexported)
func newAuthResourceHandler(service AuthResourceServiceInterface) *authResourceHandler {
    return &authResourceHandler{service: service}
}

// Handler methods
func (h *authResourceHandler) handleCreate(c *gin.Context) {
    // Move logic from auth_resource_handler.go
}

func (h *authResourceHandler) handleGet(c *gin.Context) {
    // Move logic from auth_resource_handler.go
}

// More handlers...
```

**init.go:**
```go
package authresource

import (
    "github.com/gin-gonic/gin"
    "github.com/wso2/consent-management-api/internal/system/database"
)

// Initialize sets up the auth resource module and returns service interface (EXPORTED)
func Initialize(router *gin.RouterGroup, db *database.Database) AuthResourceServiceInterface {
    // 1. Create store
    store := newAuthResourceStore(db)
    
    // 2. Create service
    service := newAuthResourceService(store)
    
    // 3. Create handler
    handler := newAuthResourceHandler(service)
    
    // 4. Register routes
    registerRoutes(router, handler)
    
    // 5. Return service for other packages to use
    return service
}

// registerRoutes registers all auth resource routes (unexported)
func registerRoutes(router *gin.RouterGroup, handler *authResourceHandler) {
    authResources := router.Group("/auth-resources")
    {
        authResources.POST("", handler.handleCreate)
        authResources.GET("/:id", handler.handleGet)
        authResources.GET("", handler.handleList)
        authResources.PUT("/:id", handler.handleUpdate)
        authResources.DELETE("/:id", handler.handleDelete)
    }
}
```

**main.go changes:**
```go
// Before
authResourceDAO := dao.NewAuthResourceDAO(db)
authResourceService := service.NewAuthResourceService(authResourceDAO)
// ... router.SetupRouter(..., authResourceService, ...)

// After
v1 := router.Group("/api/v1")
authResourceService := authresource.Initialize(v1, db)
```

---

### 4.2 Phase 2: ConsentPurpose

#### **Before (Current):**
```
internal/
├── dao/
│   ├── consent_purpose_dao.go
│   └── consent_purpose_attribute_dao.go
├── service/
│   └── consent_purpose_service.go
├── handlers/
│   └── consent_purpose_handler.go
└── models/
    ├── consent_purpose.go
    └── consent_attribute.go (shared with consent)
```

#### **After (Target):**
```
internal/
└── consentpurpose/
    ├── store.go         (combines purpose + purpose_attribute DAOs)
    ├── service.go
    ├── handler.go
    ├── init.go
    └── model/
        ├── purpose.go
        └── attribute.go
```

#### **Key Changes:**

1. **Combine Two DAOs**: consent_purpose_dao + consent_purpose_attribute_dao → single store interface
2. **Purpose Type Handlers**: Keep `purpose_type_handlers/` separate or move to `consentpurpose/validators/`?
   - **Recommendation**: Move to `consentpurpose/validators/` as it's purpose-specific validation logic

**store.go structure:**
```go
type consentPurposeStoreInterface interface {
    // Purpose operations
    CreatePurpose(...) error
    GetPurpose(id string) (*model.Purpose, error)
    ListPurposes(...) ([]*model.Purpose, error)
    UpdatePurpose(...) error
    DeletePurpose(id string) error
    
    // Attribute operations
    CreateAttribute(...) error
    GetAttributes(purposeID string) ([]*model.Attribute, error)
    UpdateAttribute(...) error
    DeleteAttribute(id string) error
}
```

---

### 4.3 Phase 3: Consent

#### **Before (Current):**
```
internal/
├── dao/
│   ├── consent_dao.go
│   ├── consent_attribute_dao.go
│   ├── consent_file_dao.go
│   └── status_audit_dao.go
├── service/
│   └── consent_service.go
├── handlers/
│   └── consent_handler.go
└── extension-client/
    └── extension_client.go
```

#### **After (Target):**
```
internal/
├── consent/
│   ├── store.go         (combines 4 DAOs: consent, attribute, file, status_audit)
│   ├── service.go
│   ├── handler.go
│   ├── init.go
│   └── model/
│       ├── consent.go
│       ├── attribute.go
│       ├── file.go
│       └── status_audit.go
└── system/
    └── extensionclient/  (move extension-client here)
```

#### **Key Changes:**

1. **Combine Four DAOs** into one store interface
2. **Dependencies**: Consent service depends on ConsentPurposeServiceInterface
3. **Extension Client**: Move to shared/extensionclient as it's cross-cutting

**store.go structure:**
```go
type consentStoreInterface interface {
    // Consent operations
    CreateConsent(...) error
    GetConsent(id string) (*model.Consent, error)
    UpdateConsent(...) error
    DeleteConsent(id string) error
    SearchConsents(...) ([]*model.Consent, error)
    
    // Attribute operations
    CreateAttributes(consentID string, attrs []*model.Attribute) error
    GetAttributes(consentID string) ([]*model.Attribute, error)
    
    // File operations
    CreateFile(...) error
    GetFiles(consentID string) ([]*model.File, error)
    
    // Status audit operations
    CreateStatusAudit(...) error
    GetStatusAudit(consentID string) ([]*model.StatusAudit, error)
}
```

**service.go dependencies:**
```go
type consentService struct {
    store           consentStoreInterface
    purposeService  consentpurpose.ConsentPurposeServiceInterface // Imported
    extensionClient *extensionclient.Client
}

func newConsentService(
    store consentStoreInterface,
    purposeService consentpurpose.ConsentPurposeServiceInterface,
    extensionClient *extensionclient.Client,
) ConsentServiceInterface {
    return &consentService{
        store:           store,
        purposeService:  purposeService,
        extensionClient: extensionClient,
    }
}
```

**main.go wiring:**
```go
// Initialize extension client (shared)
extensionClient := extensionclient.New(cfg.Extension)

// Initialize resources in dependency order
v1 := router.Group("/api/v1")
authResourceService := authresource.Initialize(v1, db)
purposeService := consentpurpose.Initialize(v1, db)
consentService := consent.Initialize(v1, db, purposeService, extensionClient)
```

---

## 5. System Components

### 5.1 Create internal/system/ for Cross-Cutting Concerns

```
internal/system/
├── database/        (move internal/database/)
├── config/          (move internal/config/)
├── extensionclient/ (move internal/extension-client/)
└── utils/           (move internal/utils/)
    ├── response_helper.go
    ├── pagination.go
    ├── validation.go
    └── context.go
```

### 5.2 Why Shared?

- **database**: Used by all resources
- **config**: Global configuration
- **extensionclient**: Used by consent (potentially others in future)
- **utils**: Common helpers (pagination, validation, response formatting)

---

## 6. Router Transition Strategy

### 6.1 Current (Centralized)

```go
// internal/router/router.go
func SetupRouter(consentService, authResourceService, purposeService, extensionClient) *gin.Engine {
    router := gin.Default()
    
    // Create all handlers
    consentHandler := handlers.NewConsentHandler(...)
    authResourceHandler := handlers.NewAuthResourceHandler(...)
    purposeHandler := handlers.NewConsentPurposeHandler(...)
    
    // Register all routes
    v1 := router.Group("/api/v1")
    v1.POST("/consents", consentHandler.CreateConsent)
    v1.GET("/consents/:id", consentHandler.GetConsent)
    // ... 50+ routes
}
```

### 6.2 Target (Distributed)

```go
// cmd/server/main.go
func main() {
    router := gin.Default()
    
    // Global middleware
    router.Use(middlewares.OrgIDExtractor())
    router.Use(middlewares.ClientIDExtractor())
    
    // Health check
    router.GET("/health", healthHandler)
    
    // Initialize extension client
    extensionClient := extensionclient.New(cfg.Extension)
    
    // Initialize resources (each registers its own routes)
    v1 := router.Group("/api/v1")
    authResourceService := authresource.Initialize(v1, db)
    purposeService := consentpurpose.Initialize(v1, db)
    consentService := consent.Initialize(v1, db, purposeService, extensionClient)
    
    // Start server
    router.Run(":8080")
}
```

### 6.3 Benefits

1. **Decentralization**: Each resource owns its routes
2. **Scalability**: Add new resources without modifying central router
3. **Clarity**: Routes defined next to handlers
4. **Testing**: Can test route registration independently

---

## 7. Testing Strategy

### 7.1 Store Tests (Unit)

```go
// internal/authresource/store_test.go
func TestAuthResourceStore_Create(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    store := newAuthResourceStore(db)
    
    resource := &model.AuthResource{...}
    err := store.Create(resource)
    
    assert.NoError(t, err)
    // Verify in database
}
```

### 7.2 Service Tests (Unit with Mock Store)

```go
// internal/authresource/service_test.go
func TestAuthResourceService_CreateAuthResource(t *testing.T) {
    mockStore := &mockAuthResourceStore{}
    service := newAuthResourceService(mockStore)
    
    resource := &model.AuthResource{...}
    result, err := service.CreateAuthResource(resource)
    
    assert.NoError(t, err)
    assert.Equal(t, resource.ID, result.ID)
}
```

### 7.3 Handler Tests (Unit with Mock Service)

```go
// internal/authresource/handler_test.go
func TestAuthResourceHandler_handleCreate(t *testing.T) {
    mockService := &mockAuthResourceService{}
    handler := newAuthResourceHandler(mockService)
    
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("POST", "/auth-resources", body)
    
    handler.handleCreate(c)
    
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

### 7.4 Integration Tests (Existing)

- Keep `tests/integration/` unchanged initially
- After refactoring, verify all 39 tests still pass
- Tests hit real HTTP endpoints, so should be unaffected by internal restructuring

---

## 8. Implementation Checklist

### Phase 1: AuthResource (Week 1)

- [ ] 1.1 Create `internal/authresource/` directory
- [ ] 1.2 Move `models/auth_resource.go` → `authresource/model/auth_resource.go`
- [ ] 1.3 Create `authresource/store.go` with interface + implementation
- [ ] 1.4 Move `service/auth_resource_service.go` → `authresource/service.go`
- [ ] 1.5 Refactor service to use store interface
- [ ] 1.6 Move `handlers/auth_resource_handler.go` → `authresource/handler.go`
- [ ] 1.7 Create `authresource/init.go` with Initialize() and registerRoutes()
- [ ] 1.8 Update `main.go` to call `authresource.Initialize()`
- [ ] 1.9 Update imports across codebase
- [ ] 1.10 Write unit tests (store, service, handler)
- [ ] 1.11 Run integration tests - verify all pass
- [ ] 1.12 Remove old files: `dao/auth_resource_dao.go`, `service/auth_resource_service.go`, `handlers/auth_resource_handler.go`

### Phase 2: ConsentPurpose (Week 2)

- [ ] 2.1 Create `internal/consentpurpose/` directory
- [ ] 2.2 Move models to `consentpurpose/model/`
- [ ] 2.3 Move `purpose_type_handlers/` → `consentpurpose/validators/`
- [ ] 2.4 Create `consentpurpose/store.go` (combine purpose + attribute DAOs)
- [ ] 2.5 Move service to `consentpurpose/service.go`
- [ ] 2.6 Move handler to `consentpurpose/handler.go`
- [ ] 2.7 Create `consentpurpose/init.go`
- [ ] 2.8 Update `main.go`
- [ ] 2.9 Update imports
- [ ] 2.10 Write tests
- [ ] 2.11 Run integration tests - verify all pass
- [ ] 2.12 Remove old files

### Phase 3: Consent (Week 3)

- [ ] 3.1 Create `internal/consent/` directory
- [ ] 3.2 Move models to `consent/model/`
- [ ] 3.3 Create `consent/store.go` (combine 4 DAOs)
- [ ] 3.4 Move service to `consent/service.go` with purposeService dependency
- [ ] 3.5 Move handler to `consent/handler.go`
- [ ] 3.6 Create `consent/init.go` with dependencies
- [ ] 3.7 Update `main.go`
- [ ] 3.8 Update imports
- [ ] 3.9 Write tests
- [ ] 3.10 Run integration tests - verify all pass
- [ ] 3.11 Remove old files

### Phase 4: Cleanup (Week 4)

- [ ] 4.1 Create `internal/system/` directory
- [ ] 4.2 Move `internal/database/` → `internal/system/database/`
- [ ] 4.3 Move `internal/config/` → `internal/system/config/`
- [ ] 4.4 Move `internal/extension-client/` → `internal/system/extensionclient/`
- [ ] 4.5 Move `internal/utils/` → `internal/system/utils/`
- [ ] 4.6 Delete `internal/dao/` (should be empty)
- [ ] 4.7 Delete `internal/service/` (should be empty)
- [ ] 4.8 Delete `internal/handlers/` (should be empty)
- [ ] 4.9 Delete `internal/models/` (should be empty)
- [ ] 4.10 Delete `internal/router/` (should be empty)
- [ ] 4.11 Update all imports to use `system/`
- [ ] 4.12 Run all tests (unit + integration)
- [ ] 4.13 Update documentation

---

## 9. Risk Mitigation

### 9.1 Potential Issues

1. **Import Cycles**: If packages depend on each other incorrectly
   - **Mitigation**: Follow dependency order (AuthResource → ConsentPurpose → Consent)

2. **Interface Compatibility**: Service interfaces must match existing usage
   - **Mitigation**: Keep service method signatures identical initially

3. **Test Breakage**: Integration tests may break during transition
   - **Mitigation**: Run tests after each phase, fix immediately

4. **Build Failures**: Import path changes across codebase
   - **Mitigation**: Use IDE refactoring tools, verify build after each step

### 9.2 Rollback Strategy

- Each phase is independent (can rollback one without affecting others)
- Use git branches: `refactor/authresource`, `refactor/consentpurpose`, `refactor/consent`
- Merge to main only after integration tests pass

---

## 10. Success Metrics

### 10.1 Code Quality
- [ ] All packages follow Thunder patterns (store, service, handler, init)
- [ ] No circular dependencies
- [ ] All interfaces defined and used correctly
- [ ] 80%+ test coverage for new code

### 10.2 Functionality
- [ ] All 39 integration tests pass
- [ ] Build succeeds: `./build.sh build`
- [ ] Start script works: `./start.sh` and `./start.sh --debug`

### 10.3 Maintainability
- [ ] Clear package boundaries
- [ ] Each resource self-contained
- [ ] Easy to add new resources
- [ ] Documentation updated (README, package docs)

---

## 11. Framework Considerations

### 11.1 Gin vs http.ServeMux

**Thunder uses:** `http.ServeMux` (Go standard library)
**Consent-server uses:** Gin framework

**Decision:** **Keep Gin for now**, adapt pattern to work with Gin

**Reasons:**
1. Gin already integrated (context, middleware, JSON binding)
2. Changing router framework is orthogonal to architecture refactoring
3. Can migrate to http.ServeMux later if desired

**Adaptation:**
```go
// Thunder: func (h *handler) Handle(w http.ResponseWriter, r *http.Request)
// Ours:    func (h *handler) Handle(c *gin.Context)

// Thunder: mux.HandleFunc("POST /applications", handler.Handle)
// Ours:    router.POST("/applications", handler.Handle)
```

### 11.2 Error Handling

**Thunder uses:** Custom `serviceerror.ServiceError` type
**Consent-server uses:** Standard Go `error`

**Decision:** **Keep standard error for now**, can enhance later

---

## 12. Next Steps

1. **Review this plan** with team
2. **Create feature branch**: `refactor/architecture`
3. **Start Phase 1**: AuthResource refactoring
4. **Daily check-ins** to monitor progress
5. **Weekly demos** showing progress

---

## Appendix A: File Mapping

### A.1 AuthResource

| Current File | New Location |
|-------------|--------------|
| `dao/auth_resource_dao.go` | `authresource/store.go` |
| `service/auth_resource_service.go` | `authresource/service.go` |
| `handlers/auth_resource_handler.go` | `authresource/handler.go` |
| `models/auth_resource.go` | `authresource/model/auth_resource.go` |
| N/A | `authresource/init.go` (new) |

### A.2 ConsentPurpose

| Current File | New Location |
|-------------|--------------|
| `dao/consent_purpose_dao.go` | `consentpurpose/store.go` |
| `dao/consent_purpose_attribute_dao.go` | `consentpurpose/store.go` |
| `service/consent_purpose_service.go` | `consentpurpose/service.go` |
| `handlers/consent_purpose_handler.go` | `consentpurpose/handler.go` |
| `models/consent_purpose.go` | `consentpurpose/model/purpose.go` |
| `models/consent_attribute.go` (shared) | `consentpurpose/model/attribute.go` |
| `purpose_type_handlers/` | `consentpurpose/validators/` |
| N/A | `consentpurpose/init.go` (new) |

### A.3 Consent

| Current File | New Location |
|-------------|--------------|
| `dao/consent_dao.go` | `consent/store.go` |
| `dao/consent_attribute_dao.go` | `consent/store.go` |
| `dao/consent_file_dao.go` | `consent/store.go` |
| `dao/status_audit_dao.go` | `consent/store.go` |
| `service/consent_service.go` | `consent/service.go` |
| `handlers/consent_handler.go` | `consent/handler.go` |
| `models/consent.go` | `consent/model/consent.go` |
| `models/consent_attribute.go` | `consent/model/attribute.go` |
| `models/consent_file.go` | `consent/model/file.go` |
| `models/status_audit.go` | `consent/model/status_audit.go` |
| N/A | `consent/init.go` (new) |

---

## Appendix B: Code Templates

### B.1 Store Template

```go
package resourcename

import "github.com/wso2/consent-management-api/internal/system/database"

// resourceStoreInterface defines data access operations (unexported)
type resourceStoreInterface interface {
    Create(item *model.Resource) error
    Get(id string) (*model.Resource, error)
    List(filters Filters) ([]*model.Resource, error)
    Update(item *model.Resource) error
    Delete(id string) error
}

// resourceStore implements the store interface
type resourceStore struct {
    db *database.Database
}

// newResourceStore creates a new store instance (unexported)
func newResourceStore(db *database.Database) resourceStoreInterface {
    return &resourceStore{db: db}
}

// Create implements resourceStoreInterface
func (s *resourceStore) Create(item *model.Resource) error {
    query := `INSERT INTO resources (id, name, ...) VALUES (?, ?, ...)`
    _, err := s.db.Exec(query, item.ID, item.Name, ...)
    return err
}

// More methods...
```

### B.2 Service Template

```go
package resourcename

// ResourceServiceInterface defines business operations (EXPORTED)
type ResourceServiceInterface interface {
    CreateResource(item *model.Resource) (*model.Resource, error)
    GetResource(id string) (*model.Resource, error)
    ListResources(filters Filters) ([]*model.Resource, error)
    UpdateResource(item *model.Resource) error
    DeleteResource(id string) error
}

// resourceService implements the service
type resourceService struct {
    store resourceStoreInterface
    // Add other service dependencies here
}

// newResourceService creates service (unexported)
func newResourceService(store resourceStoreInterface) ResourceServiceInterface {
    return &resourceService{store: store}
}

// CreateResource implements ResourceServiceInterface
func (s *resourceService) CreateResource(item *model.Resource) (*model.Resource, error) {
    // Business logic validation
    if item.Name == "" {
        return nil, errors.New("name is required")
    }
    
    // Call store
    if err := s.store.Create(item); err != nil {
        return nil, err
    }
    
    return item, nil
}

// More methods...
```

### B.3 Handler Template

```go
package resourcename

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// resourceHandler handles HTTP requests (unexported)
type resourceHandler struct {
    service ResourceServiceInterface
}

// newResourceHandler creates handler (unexported)
func newResourceHandler(service ResourceServiceInterface) *resourceHandler {
    return &resourceHandler{service: service}
}

// handleCreate handles POST /resources
func (h *resourceHandler) handleCreate(c *gin.Context) {
    var req model.Resource
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    result, err := h.service.CreateResource(&req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, result)
}

// More handlers...
```

### B.4 Init Template

```go
package resourcename

import (
    "github.com/gin-gonic/gin"
    "github.com/wso2/consent-management-api/internal/system/database"
)

// Initialize sets up the resource module and returns service interface (EXPORTED)
func Initialize(router *gin.RouterGroup, db *database.Database) ResourceServiceInterface {
    // 1. Create store
    store := newResourceStore(db)
    
    // 2. Create service
    service := newResourceService(store)
    
    // 3. Create handler
    handler := newResourceHandler(service)
    
    // 4. Register routes
    registerRoutes(router, handler)
    
    // 5. Return service for other packages to use
    return service
}

// registerRoutes registers all resource routes (unexported)
func registerRoutes(router *gin.RouterGroup, handler *resourceHandler) {
    resources := router.Group("/resources")
    {
        resources.POST("", handler.handleCreate)
        resources.GET("/:id", handler.handleGet)
        resources.GET("", handler.handleList)
        resources.PUT("/:id", handler.handleUpdate)
        resources.DELETE("/:id", handler.handleDelete)
    }
}
```

---

## 13. Thunder System Package Analysis

### 13.1 Overview

The Thunder `internal/system/` package provides **robust common utilities** that should be adopted in consent-server. This includes:

1. **Database handling** with transaction support
2. **Standardized error responses** (API + Service layers)
3. **Middleware** (Correlation ID, CORS)
4. **Constants** for HTTP headers, pagination, etc.
5. **Utility functions** for HTTP, strings, JSON handling

### 13.2 Database Architecture

#### **Multi-Database Support**

Thunder supports **MySQL, PostgreSQL, and SQLite** with database-specific query handling:

```go
// DBQuery with database-specific variants
type DBQuery struct {
    ID            string  // Unique query identifier
    Query         string  // Default MySQL query
    PostgresQuery string  // PostgreSQL-specific
    SQLiteQuery   string  // SQLite-specific
}

// GetQuery returns appropriate query for database type
func (d *DBQuery) GetQuery(dbType string) string {
    switch dbType {
    case "postgres":
        return d.PostgresQuery // Use Postgres variant if available
    case "sqlite":
        return d.SQLiteQuery   // Use SQLite variant if available
    default:
        return d.Query         // Fallback to default (MySQL)
    }
}
```

**Example Usage:**
```go
var QueryGetApplication = model.DBQuery{
    ID:            "GET_APPLICATION",
    Query:         "SELECT * FROM applications WHERE id = ?",           // MySQL
    PostgresQuery: "SELECT * FROM applications WHERE id = $1",          // PostgreSQL ($1)
    SQLiteQuery:   "SELECT * FROM applications WHERE id = ?",           // SQLite
}
```

#### **Transaction Pattern (Multiple Queries)**

Thunder uses a **functional transaction pattern** for executing multiple queries atomically:

```go
// Define queries as functions that accept transaction
queries := []func(tx dbmodel.TxInterface) error{
    func(tx dbmodel.TxInterface) error {
        _, err := tx.Exec(QueryInsertApplication.GetQuery(dbType), appID, name, ...)
        return err
    },
    func(tx dbmodel.TxInterface) error {
        _, err := tx.Exec(QueryInsertOAuthConfig.GetQuery(dbType), clientID, secret, ...)
        return err
    },
    func(tx dbmodel.TxInterface) error {
        _, err := tx.Exec(QueryInsertCertificate.GetQuery(dbType), appID, cert, ...)
        return err
    },
}

// Execute all queries in a single transaction
if err := executeTransaction(queries); err != nil {
    return err
}
```

**Transaction Helper:**
```go
func executeTransaction(queries []func(tx dbmodel.TxInterface) error) error {
    tx, err := dbClient.BeginTx()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    // Execute each query
    for _, query := range queries {
        if err := query(tx); err != nil {
            // Rollback on any error
            if rollbackErr := tx.Rollback(); rollbackErr != nil {
                return errors.Join(err, rollbackErr) // Combine errors
            }
            return err
        }
    }

    // Commit if all succeed
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

**Benefits:**
- ✅ **Atomic operations**: All queries succeed or all fail
- ✅ **Composable**: Build complex transactions from simple query functions
- ✅ **Automatic rollback**: No manual cleanup needed
- ✅ **Error aggregation**: Uses `errors.Join()` to preserve all error context

#### **JSON Query Builder**

Thunder provides utilities for building dynamic JSON queries (for JSON columns):

```go
// Build filter query for JSON column
query, args, err := queryutils.BuildFilterQuery(
    "FILTER_APPLICATIONS",
    "SELECT * FROM applications WHERE 1=1",
    "config",  // JSON column name
    map[string]interface{}{
        "email": "user@example.com",
        "address.city": "Colombo",  // Nested JSON path
    },
)

// Results in:
// PostgreSQL: SELECT * FROM applications WHERE 1=1 AND config->>'email' = $1 AND config#>>'{address,city}' = $2
// SQLite:     SELECT * FROM applications WHERE 1=1 AND json_extract(config, '$.email') = ? AND json_extract(config, '$.address.city') = ?
```

**Key Features:**
- **Nested JSON paths**: `address.city` → `config#>>'{address,city}'` (Postgres) or `json_extract(config, '$.address.city')` (SQLite)
- **SQL injection prevention**: Validates keys contain only alphanumeric + underscore + dot
- **Parameterized queries**: Uses `$1, $2` (Postgres) or `?` (MySQL/SQLite)

### 13.3 Error Handling Architecture

Thunder uses **two-tier error system**: Service errors + API errors

#### **Service Errors (Internal)**

```go
// ServiceError for business logic layer
type ServiceError struct {
    Code             string           `json:"code"`
    Type             ServiceErrorType `json:"type"`  // "client_error" or "server_error"
    Error            string           `json:"error"`
    ErrorDescription string           `json:"error_description,omitempty"`
}

// Predefined errors
var (
    InternalServerError = ServiceError{
        Type:             ServerErrorType,
        Code:             "SSE-5000",
        Error:            "Internal server error",
        ErrorDescription: "An unexpected error occurred",
    }
    
    InvalidRequestError = ServiceError{
        Type:             ClientErrorType,
        Code:             "CSE-4000",
        Error:            "Invalid request",
        ErrorDescription: "The request is malformed or invalid",
    }
)

// Custom error with dynamic description
func CustomServiceError(baseError ServiceError, desc string) *ServiceError {
    return &ServiceError{
        Type:             baseError.Type,
        Code:             baseError.Code,
        Error:            baseError.Error,
        ErrorDescription: desc,  // Override description
    }
}
```

#### **API Errors (HTTP Responses)**

```go
// ErrorResponse for API responses
type ErrorResponse struct {
    Code        string `json:"code"`
    Message     string `json:"message"`
    Description string `json:"description"`
}

// Utility to write JSON error response
func WriteJSONError(w http.ResponseWriter, code, desc string, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{
        "error":             code,
        "error_description": desc,
    })
}
```

#### **Error Flow**

```
Service Layer             Handler Layer             HTTP Response
────────────────         ────────────────         ────────────────
ServiceError       →     Check error type    →    HTTP Status + JSON
(Code: CSE-4001)         (ClientError?)           {
(Type: client_error)     Yes → 400/404/409          "error": "invalid_input",
                                                     "error_description": "..."
                         (ServerError?)           }
                         Yes → 500/503
```

**Example:**
```go
// Service layer
func (s *service) CreateConsent(consent *model.Consent) (*model.Consent, *serviceerror.ServiceError) {
    if consent.UserID == "" {
        return nil, &serviceerror.InvalidRequestError  // Return service error
    }
    // ...
}

// Handler layer
func (h *handler) handleCreate(w http.ResponseWriter, r *http.Request) {
    result, svcErr := h.service.CreateConsent(consent)
    if svcErr != nil {
        statusCode := 500
        if svcErr.Type == serviceerror.ClientErrorType {
            statusCode = 400  // Map client errors to 4xx
        }
        utils.WriteJSONError(w, svcErr.Error, svcErr.ErrorDescription, statusCode)
        return
    }
    // Success response
    utils.JSONResponse(w, http.StatusCreated, result)
}
```

**Benefits:**
- ✅ **Separation of concerns**: Service doesn't know about HTTP
- ✅ **Consistent format**: All errors follow same structure
- ✅ **Type safety**: ServiceError vs generic error
- ✅ **Reusable**: Predefined errors reduce duplication

### 13.4 Middleware: Correlation ID

Thunder implements **correlation ID (trace ID)** for request tracking:

```go
// CorrelationIDMiddleware extracts or generates correlation ID
func CorrelationIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Try to extract from headers
        correlationID := extractCorrelationID(r)  // Checks X-Correlation-ID, X-Request-ID, X-Trace-ID
        
        // 2. Generate new UUID if not present
        if correlationID == "" {
            correlationID = uuid.New().String()
        }
        
        // 3. Store in request context
        ctx := context.WithValue(r.Context(), "trace_id", correlationID)
        r = r.WithContext(ctx)
        
        // 4. Add to response headers
        w.Header().Set("X-Correlation-ID", correlationID)
        
        // 5. Continue to next handler
        next.ServeHTTP(w, r)
    })
}

// Extract from common header names
func extractCorrelationID(r *http.Request) string {
    headers := []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"}
    for _, header := range headers {
        if id := r.Header.Get(header); id != "" {
            return id
        }
    }
    return ""
}
```

**Usage in logs:**
```go
// Logger automatically includes trace ID from context
logger := log.GetLogger()  // Gets trace ID from context
logger.Info("Processing request",
    log.String("user_id", userID),
    log.String("trace_id", ctx.Value("trace_id").(string)),  // Included automatically
)
```

**Benefits:**
- ✅ **End-to-end tracing**: Same ID across logs, responses, downstream calls
- ✅ **Debugging**: Search logs by correlation ID to see entire request flow
- ✅ **Client tracking**: Clients can track their requests using returned X-Correlation-ID
- ✅ **Distributed systems**: Pass correlation ID to microservices for distributed tracing

### 13.5 Middleware: CORS

Thunder implements **configurable CORS** with per-route options:

```go
// CORSOptions for fine-grained control
type CORSOptions struct {
    AllowedMethods   string  // "GET, POST, PUT"
    AllowedHeaders   string  // "Content-Type, Authorization"
    AllowCredentials bool    // true/false
}

// WithCORS wraps handler with CORS headers
func WithCORS(pattern string, handler http.HandlerFunc, opts CORSOptions) (string, http.HandlerFunc) {
    return pattern, func(w http.ResponseWriter, r *http.Request) {
        applyCORSHeaders(w, r, opts)  // Apply CORS first
        handler(w, r)                  // Then call handler
    }
}

// Apply CORS headers based on config
func applyCORSHeaders(w http.ResponseWriter, r *http.Request, opts CORSOptions) {
    requestOrigin := r.Header.Get("Origin")
    if requestOrigin == "" {
        return  // No CORS needed if no Origin header
    }
    
    allowedOrigins := config.GetCORSConfig().AllowedOrigins  // From deployment.yaml
    
    // Check if request origin is allowed
    if allowedOrigin := matchOrigin(allowedOrigins, requestOrigin); allowedOrigin != "" {
        w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
        w.Header().Set("Access-Control-Allow-Methods", opts.AllowedMethods)
        w.Header().Set("Access-Control-Allow-Headers", opts.AllowedHeaders)
        if opts.AllowCredentials {
            w.Header().Set("Access-Control-Allow-Credentials", "true")
        }
    }
}
```

**Route registration with CORS:**
```go
func registerRoutes(mux *http.ServeMux, handler *handler) {
    opts1 := middleware.CORSOptions{
        AllowedMethods:   "GET, POST",
        AllowedHeaders:   "Content-Type, Authorization",
        AllowCredentials: true,
    }
    
    mux.HandleFunc(middleware.WithCORS("POST /consents", handler.handleCreate, opts1))
    mux.HandleFunc(middleware.WithCORS("GET /consents", handler.handleList, opts1))
    
    opts2 := middleware.CORSOptions{
        AllowedMethods:   "GET, PUT, DELETE",
        AllowedHeaders:   "Content-Type, Authorization",
        AllowCredentials: true,
    }
    
    mux.HandleFunc(middleware.WithCORS("GET /consents/{id}", handler.handleGet, opts2))
    mux.HandleFunc(middleware.WithCORS("PUT /consents/{id}", handler.handleUpdate, opts2))
}
```

**Configuration (deployment.yaml):**
```yaml
cors:
  allowed_origins:
    - "http://localhost:3000"
    - "https://app.example.com"
    - "*"  # Allow all (not recommended for production)
```

**Benefits:**
- ✅ **Per-route configuration**: Different CORS settings for different endpoints
- ✅ **Origin validation**: Only configured origins allowed
- ✅ **Credentials support**: Enable/disable per route
- ✅ **Centralized config**: Origins managed in deployment.yaml

### 13.6 Constants Package

Thunder centralizes all constants:

```go
package constants

// HTTP Headers
const (
    AuthorizationHeaderName = "Authorization"
    ContentTypeHeaderName   = "Content-Type"
    AcceptHeaderName        = "Accept"
    CacheControlHeaderName  = "Cache-Control"
)

// Content Types
const (
    ContentTypeJSON           = "application/json"
    ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

// Cache Control
const (
    CacheControlNoStore = "no-store"
    PragmaNoCache       = "no-cache"
)

// Pagination
const (
    DefaultPageSize = 30
    MaxPageSize     = 100
)

// Authentication
const (
    TokenTypeBearer  = "Bearer"
    AuthSchemeBasic  = "Basic "
)
```

**Benefits:**
- ✅ **Single source of truth**: All constants in one place
- ✅ **Type safety**: Constants instead of magic strings
- ✅ **Easy updates**: Change once, applies everywhere
- ✅ **Discoverability**: Developers know where to look

### 13.7 HTTP Utilities

Thunder provides robust HTTP utilities:

```go
// DecodeJSONBody - Generic JSON decoding
func DecodeJSONBody[T any](r *http.Request) (*T, error) {
    var data T
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("failed to decode JSON: %w", err)
    }
    return &data, nil
}

// Usage:
request, err := DecodeJSONBody[model.ConsentRequest](r)

// JSONResponse - Standard JSON response
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(data)
}

// SanitizeString - Security: Remove control chars, escape HTML
func SanitizeString(input string) string {
    // 1. Trim whitespace
    trimmed := strings.TrimSpace(input)
    
    // 2. Remove control characters (except newline/tab)
    cleaned := strings.Map(func(r rune) rune {
        if unicode.IsControl(r) && r != '\n' && r != '\t' {
            return -1  // Remove
        }
        return r
    }, trimmed)
    
    // 3. Escape HTML to prevent XSS
    safe := html.EscapeString(cleaned)
    
    return safe
}

// IsValidURI - Validate URIs
func IsValidURI(uri string) bool {
    parsed, err := url.Parse(uri)
    return err == nil && parsed.Scheme != "" && parsed.Host != ""
}
```

### 13.8 Adoption Plan for Consent Server

#### **Phase 0: System Infrastructure (Before Resource Refactoring)**

Create `internal/system/` with Thunder patterns:

```
internal/system/
├── database/
│   ├── model/
│   │   ├── dbquery.go        (DBQuery with multi-DB support)
│   │   ├── model.go          (DBInterface, TxInterface)
│   │   └── transaction.go    (executeTransaction helper)
│   ├── provider/
│   │   └── dbclient.go       (DBClient with Query/Execute/BeginTx)
│   └── utils/
│       └── querybuilder.go   (BuildFilterQuery for JSON columns)
├── error/
│   ├── apierror/
│   │   └── error.go          (ErrorResponse)
│   └── serviceerror/
│       └── error.go          (ServiceError with types)
├── middleware/
│   ├── correlationid.go      (CorrelationIDMiddleware)
│   └── cors.go               (WithCORS wrapper)
├── constants/
│   └── constants.go          (HTTP headers, content types, pagination)
└── utils/
    ├── httputil.go           (DecodeJSONBody, JSONResponse, WriteJSONError)
    └── security.go           (SanitizeString, IsValidURI)
```

#### **Migration Steps**

1. **Create system/database package** (Week 1)
   - Implement DBQuery with multi-DB support
   - Add transaction helper
   - Update existing database.Database to use new patterns
   - Keep backward compatibility initially

2. **Create system/error package** (Week 1)
   - Define ServiceError types
   - Define ErrorResponse
   - Create utility functions
   - Don't migrate existing code yet

3. **Create system/middleware package** (Week 2)
   - Implement CorrelationIDMiddleware
   - Implement CORS middleware (adapt for Gin)
   - Add to main.go router setup

4. **Create system/constants + utils** (Week 2)
   - Move existing constants
   - Add HTTP utilities
   - Add security utilities

5. **Integrate during resource refactoring** (Week 3-4)
   - AuthResource: Use new store patterns + ServiceError
   - ConsentPurpose: Use transaction patterns
   - Consent: Full adoption of all system utilities

#### **Gin Adaptation**

Since consent-server uses **Gin**, adapt Thunder patterns:

**Middleware:**
```go
// Thunder: func(next http.Handler) http.Handler
// Gin:    func(c *gin.Context)

func CorrelationIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        correlationID := extractCorrelationID(c.Request)
        if correlationID == "" {
            correlationID = uuid.New().String()
        }
        
        // Store in Gin context
        c.Set("trace_id", correlationID)
        
        // Add to response
        c.Header("X-Correlation-ID", correlationID)
        
        c.Next()
    }
}

// Usage in main.go
router.Use(CorrelationIDMiddleware())
```

**Error Response:**
```go
// Service returns ServiceError
result, svcErr := h.service.CreateConsent(consent)
if svcErr != nil {
    statusCode := http.StatusInternalServerError
    if svcErr.Type == serviceerror.ClientErrorType {
        statusCode = http.StatusBadRequest
    }
    c.JSON(statusCode, gin.H{
        "error":             svcErr.Error,
        "error_description": svcErr.ErrorDescription,
    })
    return
}
c.JSON(http.StatusCreated, result)
```

---

## Summary

This refactoring transforms consent-server from a **traditional layered architecture** to a **modern resource-based architecture** following industry best practices demonstrated by the Thunder project.

**Key Benefits:**
1. ✅ Better code organization (package-per-resource)
2. ✅ Improved testability (interface-based design)
3. ✅ Easier maintenance (colocated related code)
4. ✅ Clear boundaries (each resource self-contained)
5. ✅ Scalable structure (easy to add new resources)
6. ✅ **Robust database handling** (transactions, multi-DB support)
7. ✅ **Standardized error handling** (service + API layers)
8. ✅ **Production-ready middleware** (correlation ID, CORS)
9. ✅ **Security utilities** (input sanitization, validation)

**Estimated Timeline:** 4 weeks
- Week 1: Shared infrastructure (database, errors)
- Week 2: Middleware + constants + AuthResource refactoring
- Week 3: ConsentPurpose refactoring
- Week 4: Consent refactoring + cleanup

**Risk Level:** Medium (requires careful migration but incremental approach reduces risk)
**Success Criteria:** All tests pass, build works, architecture matches Thunder patterns

---

## Implementation Progress

### ✅ Phase 0: System Infrastructure (COMPLETED)
- ✅ Created `internal/system/database/provider` - DBClient with multi-DB support
- ✅ Created `internal/system/error/serviceerror` - ServiceError with 2-tier error handling
- ✅ Created `internal/system/middleware` - CorrelationID & CORS for http.ServeMux
- ✅ Created `internal/system/constants` - HTTP headers and common constants
- ✅ Updated `internal/utils` - Validation, time utils, pagination, response helpers

### ✅ Phase 1: AuthResource (COMPLETED)
- ✅ Created package structure: `internal/authresource/`
- ✅ Migrated models to `internal/authresource/model/`
- ✅ Created `store.go` with Thunder pattern (10 DBQuery objects, single method interface)
- ✅ Created `service.go` with ServiceError returns (9 service methods)
- ✅ Created `handler.go` with http.ServeMux (9 HTTP handlers)
- ✅ Created `init.go` with Initialize() function
- ✅ Wired into `main.go` with http.ServeMux + hybrid Gin approach
- ✅ **Build Status:** ✅ Compiles successfully

### ✅ Phase 2: ConsentPurpose (COMPLETED)
- ✅ Created package structure: `internal/consentpurpose/`
- ✅ Migrated models to `internal/consentpurpose/model/`
- ✅ Copied validators to `internal/consentpurpose/validators/`
- ✅ Created `store.go` with Thunder pattern (12 DBQuery objects, combines purpose + attribute DAOs)
- ✅ Created `service.go` with ServiceError returns (5 service methods + validation)
- ✅ Created `handler.go` with http.ServeMux (5 HTTP handlers)
- ✅ Created `init.go` with Initialize() function (5 routes registered)
- ✅ Wired into `main.go` after AuthResource
- ✅ **Build Status:** ✅ Compiles successfully

### ⏳ Phase 3: Consent (PENDING)
- ⏳ Create package structure: `internal/consent/`
- ⏳ Migrate models (consent, consent_attribute, consent_file, status_audit)
- ⏳ Create store.go (combines 4 DAOs)
- ⏳ Create service.go with extension client integration
- ⏳ Create handler.go with http.ServeMux
- ⏳ Create init.go
- ⏳ Wire into main.go
- ⏳ Test and verify

### ⏳ Phase 4: Cleanup (PENDING)
- ⏳ Remove old dao/ directory
- ⏳ Remove old handlers/ directory
- ⏳ Remove old service/ directory
- ⏳ Remove old models/ directory (if fully migrated)
- ⏳ Remove Gin router and dependencies
- ⏳ Update all imports
- ⏳ Run full test suite
- ⏳ Update documentation

**Current Status:** Phase 2 Complete - 2 of 3 resources migrated (66% complete)  
**Next Step:** Begin Phase 3 - Consent resource refactoring

---

## TODO: Swagger API Compliance

### Remaining Endpoints to Implement

**Priority: High**
- [ ] **GET /consents/attributes** - Search consents by attribute key/value
  - Query params: `key` (required), `value` (optional)
  - Returns: List of consent IDs matching the attribute criteria
  
- [ ] **POST /consents/validate** - Validate consent for authorization
  - Validates if a consent is valid for a specific authorization request
  - Used by resource servers to verify consent validity

- [ ] **POST /consent-purposes/validate** - Validate purpose type/value
  - Validates purpose attribute values against registered type handlers
  - Used during consent creation to ensure purpose data integrity

**Priority: Medium**
- [ ] Review and remove DELETE /consents/{consentId} if not in swagger spec

### API Path Corrections Completed ✅
- ✅ Fixed consent-purpose paths: `/purposes` → `/consent-purposes`
- ✅ Fixed authorization paths: `/auth-resources` → `/authorizations`
- ✅ Fixed path parameters: `{id}` → `{consentId}`, `{authId}` → `{authorizationId}`, `{id}` → `{purposeId}`
- ✅ Fixed revoke endpoint: `PATCH /consents/{id}/status` → `POST /consents/{consentId}/revoke`

### Request/Response Schema Validation
- [ ] Compare implemented request/response models with swagger definitions
- [ ] Ensure header names match exactly (org-id, TPP-client-id, etc.)
- [ ] Validate error response format matches swagger
- [ ] Verify pagination parameters and response structure
- [ ] Check query parameter naming conventions

---

**Author:** GitHub Copilot  
**Date:** December 2025  
**Version:** 1.3
