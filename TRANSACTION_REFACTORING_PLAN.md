# Transaction Refactoring Plan: Thunder Pattern Implementation

## Overview
Replace duplicate methods (`Create` + `CreateWithTx`) with single transactional methods. All store methods will work with `dbmodel.TxInterface`, and services will always use `executeTransaction` helper.

**Goal:** Remove ~500 lines of duplicate code while maintaining Thunder's functional composition pattern.

---

## Architectural Decision: Cross-Module Dependencies

### Why We Keep Exported Interfaces

**Question:** Why does `consent/service.go` need to redefine `AuthResourceStore` and `ConsentPurposeStore` interfaces?

**Answer:** To enable atomic cross-module transactions while avoiding circular dependencies.

### The Problem
```
consent package needs → authresource.store (from authresource package)
authresource package needs → consent.something (circular dependency!)
```

### Solution: Minimal Exported Interfaces in Consent Package

```go
// consent/service.go

// AuthResourceStore - minimal interface for cross-module transactions
type AuthResourceStore interface {
    Create(tx dbmodel.TxInterface, authResource *authmodel.AuthResource) error
    UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
}

// ConsentPurposeStore - minimal interface for cross-module transactions
type ConsentPurposeStore interface {
    LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}
```

### Why This is Necessary

**Option A: Service Orchestration (What we use)** ✅
- Consent service depends on other stores directly via minimal interfaces
- ✅ Simple, direct
- ✅ ACID guarantees for cross-module operations
- ✅ Single transaction for consent + auth resources + purposes
- ⚠️ Requires exported interfaces (but minimal)
- ⚠️ Some coupling (but only at service layer)

**Option B: Pure Thunder Pattern - No Cross-Store Dependencies** ❌
- Each service only manages its own store
- Cross-module operations via:
  1. **Event-driven**: Consent service emits event, authresource service listens
  2. **Two-phase**: Create consent first, then call authresource service
  3. **Database triggers**: DB handles related records
- ❌ No ACID guarantees across modules
- ❌ Complex error handling
- ❌ Not suitable for our use case

**Option C: Store Registry Pattern (Centralized Store Access)** ⭐⭐⭐ **RECOMMENDED**

Create a single registry object that holds all stores. All services share access to this registry.

```go
// internal/system/stores/registry.go
package stores

import (
    "github.com/wso2/consent-management-api/internal/system/database/provider"
    dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
)

// StoreRegistry holds references to all stores in the application
// No need to define interfaces here - just hold the actual stores!
type StoreRegistry struct {
    dbClient provider.DBClientInterface
    
    // Store instances (not interfaces) - each module defines its own interface
    Consent        interface{}  // Will be consent.consentStore
    AuthResource   interface{}  // Will be authresource.authResourceStore
    ConsentPurpose interface{}  // Will be consentpurpose.consentPurposeStore
}

// NewStoreRegistry creates a new store registry with all initialized stores
func NewStoreRegistry(
    dbClient provider.DBClientInterface,
    consentStore interface{},
    authResourceStore interface{},
    consentPurposeStore interface{},
) *StoreRegistry {
    return &StoreRegistry{
        dbClient:       dbClient,
        Consent:        consentStore,
        AuthResource:   authResourceStore,
        ConsentPurpose: consentPurposeStore,
    }
}

// ExecuteTransaction executes multiple store operations in a single transaction
func (r *StoreRegistry) ExecuteTransaction(queries []func(tx dbmodel.TxInterface) error) error {
    tx, err := r.dbClient.BeginTx()
    if err != nil {
        return err
    }
    
    for _, query := range queries {
        if err := query(tx); err != nil {
            tx.Rollback()
            return err
        }
    }
    
    return tx.Commit()
}
```

**Usage in services:**

```go
// consent/service.go
type consentService struct {
    stores *stores.StoreRegistry  // ✅ Access to ALL stores
}

func (s *consentService) CreateConsent(ctx, req, clientID, orgID) (*model.ConsentResponse, error) {
    // Type assert to get the actual store with its methods
    consentStore := s.stores.Consent.(consentStore)
    authResourceStore := s.stores.AuthResource.(authResourceStore)
    
    // Build queries using stores - call methods defined in their own interfaces
    queries := []func(tx dbmodel.TxInterface) error{
        func(tx dbmodel.TxInterface) error {
            return consentStore.Create(tx, consent)
        },
        func(tx dbmodel.TxInterface) error {
            return consentStore.CreateAttributes(tx, attributes)
        },
        func(tx dbmodel.TxInterface) error {
            return authResourceStore.Create(tx, authResource)  // ✅ Direct access!
        },
    }
    
    return s.stores.ExecuteTransaction(queries)
}

// authresource/service.go
type authResourceService struct {
    stores *stores.StoreRegistry  // ✅ Same registry!
}

func (s *authResourceService) UpdateAuthStatus(ctx, authID, status) error {
    // Type assert to get actual stores
    authResourceStore := s.stores.AuthResource.(authResourceStore)
    consentStore := s.stores.Consent.(consentStore)
    
    queries := []func(tx dbmodel.TxInterface) error{
        func(tx dbmodel.TxInterface) error {
            return authResourceStore.UpdateStatus(tx, authID, status)
        },
        // ✅ Can also update consent if needed!
        func(tx dbmodel.TxInterface) error {
            return consentStore.UpdateStatus(tx, consentID, newStatus)
        },
    }
    
    return s.stores.ExecuteTransaction(queries)
}
```

**Initialization in servicemanager:**

```go
// cmd/server/servicemanager.go
func registerServices(mux *http.ServeMux, dbClient provider.DBClientInterface) {
    // Create all stores
    consentStore := consent.NewStore(dbClient)
    authResourceStore := authresource.NewStore(dbClient)
    consentPurposeStore := consentpurpose.NewStore(dbClient)
    
    // Create registry with all stores
    storeRegistry := stores.NewStoreRegistry(
        dbClient,
        consentStore,
        authResourceStore,
        consentPurposeStore,
    )
    
    // Pass registry to all services - no circular dependencies!
    consentService = consent.Initialize(mux, storeRegistry)
    authResourceService = authresource.Initialize(mux, storeRegistry)
    consentPurposeService = consentpurpose.Initialize(mux, storeRegistry)
}
```

**Benefits:**
- ✅ **No circular dependencies** - Registry is neutral, holds interface{} references
- ✅ **No interface duplication** - Each store keeps its own interface in its package
- ✅ **All services equal** - Any service can access any store
- ✅ **Easy to extend** - Add new stores to registry
- ✅ **Clear architecture** - Stores are pure data access, services are logic
- ✅ **ACID guarantees** - Transaction helper in registry
- ✅ **Minimal coupling** - Registry doesn't need to know store internals

**Drawbacks:**
- ⚠️ Type assertions needed (but only once per service method)
- ⚠️ Services have access to all stores (but that's the point!)

**Why This is Better Than Option A:**
- No need to define minimal interfaces in each service package
- No need to pass multiple store dependencies individually
- No interface duplication - each store interface stays in its own package
- Future-proof - adding cross-module operations is trivial
- Cleaner imports - registry doesn't import all domain packages

---

**Conclusion:** **Option C (Store Registry)** is the best approach for your use case. It solves the circular dependency problem elegantly while maintaining ACID guarantees and clean architecture.

## File-by-File Changes

**UPDATE:** With Store Registry pattern, the refactoring changes slightly:

### New File: `/consent-server/internal/system/stores/registry.go`

Create this new file with the `StoreRegistry` implementation shown above.

### Modified Approach for Services

Instead of:
```go
// consent/service.go - OLD
type consentService struct {
    store               consentStore
    authResourceStore   AuthResourceStore     // ❌ Remove
    consentPurposeStore ConsentPurposeStore   // ❌ Remove
    dbClient            provider.DBClientInterface
}
```

Use:
```go
// consent/service.go - NEW with Registry
type consentService struct {
    stores *stores.StoreRegistry  // ✅ Single dependency!
}
```

All store method calls become:
```go
// OLD
s.store.Create(tx, consent)
s.authResourceStore.Create(tx, authResource)

// NEW
s.stores.Consent.Create(tx, consent)
s.stores.AuthResource.Create(tx, authResource)
```

Transaction execution:
```go
// OLD
executeTransaction(s.dbClient, queries)

// NEW
s.stores.ExecuteTransaction(queries)
```

---

## Original File-by-File Changes (with Registry adjustments)

### File 1: `/consent-server/internal/consent/store.go`

#### Changes Summary
- Remove duplicate interface methods (`Create` vs `CreateWithTx`)
- All methods accept `tx dbmodel.TxInterface` as first parameter
- Remove all old non-transactional implementations
- Remove old `WithTx` methods
- Add `executeTransaction` helper

#### Interface Changes

**BEFORE (lines 82-103):**
```go
type consentStore interface {
    // Non-transactional methods
    Create(ctx context.Context, consent *model.Consent) error
    GetByID(ctx context.Context, consentID, orgID string) (*model.Consent, error)
    List(ctx context.Context, orgID string, limit, offset int) ([]model.Consent, int, error)
    Update(ctx context.Context, consent *model.Consent) error
    UpdateStatus(ctx context.Context, consentID, orgID, status string, updatedTime int64) error
    Delete(ctx context.Context, consentID, orgID string) error
    GetByClientID(ctx context.Context, clientID, orgID string) ([]model.Consent, error)
    
    CreateAttributes(ctx context.Context, attributes []model.ConsentAttribute) error
    GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error)
    DeleteAttributesByConsentID(ctx context.Context, consentID, orgID string) error
    
    CreateStatusAudit(ctx context.Context, audit *model.ConsentStatusAudit) error
    GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentStatusAudit, error)
    
    // Duplicate transactional methods ❌
    CreateWithTx(ctx context.Context, tx *database.Tx, consent *model.Consent) error
    CreateAttributesWithTx(ctx context.Context, tx *database.Tx, attributes []model.ConsentAttribute) error
    CreateStatusAuditWithTx(ctx context.Context, tx *database.Tx, audit *model.ConsentStatusAudit) error
}
```

**AFTER:**
```go
type consentStore interface {
    // All methods are transactional - no duplication ✅
    Create(tx dbmodel.TxInterface, consent *model.Consent) error
    GetByID(tx dbmodel.TxInterface, consentID, orgID string) (*model.Consent, error)
    List(tx dbmodel.TxInterface, orgID string, limit, offset int) ([]model.Consent, int, error)
    Update(tx dbmodel.TxInterface, consent *model.Consent) error
    UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
    Delete(tx dbmodel.TxInterface, consentID, orgID string) error
    GetByClientID(tx dbmodel.TxInterface, clientID, orgID string) ([]model.Consent, error)
    
    CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error
    GetAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) ([]model.ConsentAttribute, error)
    DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
    
    CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error
    GetStatusAuditByConsentID(tx dbmodel.TxInterface, consentID, orgID string) ([]model.ConsentStatusAudit, error)
}
```

#### Implementation Examples

**Delete old implementations (lines 113-281):**
- Remove all methods like `Create(ctx context.Context, ...)`, `GetByID(ctx context.Context, ...)`, etc.

**Replace with transactional versions:**

```go
// Create creates a consent within a transaction
func (s *store) Create(tx dbmodel.TxInterface, consent *model.Consent) error {
    _, err := tx.Exec(QueryCreateConsent.Query,
        consent.ConsentID, consent.CreatedTime, consent.UpdatedTime, consent.ClientID,
        consent.ConsentType, consent.CurrentStatus, consent.ConsentFrequency,
        consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
        consent.OrgID)
    return err
}

// GetByID retrieves a consent within a transaction
func (s *store) GetByID(tx dbmodel.TxInterface, consentID, orgID string) (*model.Consent, error) {
    rows, err := tx.Query(QueryGetConsentByID.Query, consentID, orgID)
    if err != nil {
        return nil, err
    }
    if len(rows) == 0 {
        return nil, nil
    }
    return mapToConsent(rows[0]), nil
}

// List retrieves paginated consents within a transaction
func (s *store) List(tx dbmodel.TxInterface, orgID string, limit, offset int) ([]model.Consent, int, error) {
    countRows, err := tx.Query(QueryCountConsents.Query, orgID)
    if err != nil {
        return nil, 0, err
    }
    
    totalCount := 0
    if len(countRows) > 0 {
        if count, ok := countRows[0]["count"].(int64); ok {
            totalCount = int(count)
        }
    }
    
    rows, err := tx.Query(QueryListConsents.Query, orgID, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    
    consents := make([]model.Consent, 0, len(rows))
    for _, row := range rows {
        consent := mapToConsent(row)
        if consent != nil {
            consents = append(consents, *consent)
        }
    }
    
    return consents, totalCount, nil
}

// CreateAttributes creates multiple consent attributes within a transaction
func (s *store) CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error {
    for _, attr := range attributes {
        _, err := tx.Exec(QueryCreateAttribute.Query,
            attr.ConsentID, attr.AttKey, attr.AttValue, attr.OrgID)
        if err != nil {
            return err
        }
    }
    return nil
}

// CreateStatusAudit creates a status audit entry within a transaction
func (s *store) CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error {
    _, err := tx.Exec(QueryCreateStatusAudit.Query,
        audit.StatusAuditID, audit.ConsentID, audit.CurrentStatus, audit.ActionTime,
        audit.Reason, audit.ActionBy, audit.PreviousStatus, audit.OrgID)
    return err
}

// Update updates a consent within a transaction
func (s *store) Update(tx dbmodel.TxInterface, consent *model.Consent) error {
    _, err := tx.Exec(QueryUpdateConsent.Query,
        consent.UpdatedTime, consent.ConsentType, consent.ConsentFrequency,
        consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
        consent.ConsentID, consent.OrgID)
    return err
}

// UpdateStatus updates consent status within a transaction
func (s *store) UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error {
    _, err := tx.Exec(QueryUpdateConsentStatus.Query, status, updatedTime, consentID, orgID)
    return err
}

// Delete deletes a consent within a transaction
func (s *store) Delete(tx dbmodel.TxInterface, consentID, orgID string) error {
    _, err := tx.Exec(QueryDeleteConsent.Query, consentID, orgID)
    return err
}

// GetByClientID retrieves consents by client ID within a transaction
func (s *store) GetByClientID(tx dbmodel.TxInterface, clientID, orgID string) ([]model.Consent, error) {
    rows, err := tx.Query(QueryGetConsentsByClientID.Query, clientID, orgID)
    if err != nil {
        return nil, err
    }
    
    consents := make([]model.Consent, 0, len(rows))
    for _, row := range rows {
        consent := mapToConsent(row)
        if consent != nil {
            consents = append(consents, *consent)
        }
    }
    
    return consents, nil
}

// GetAttributesByConsentID retrieves attributes within a transaction
func (s *store) GetAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) ([]model.ConsentAttribute, error) {
    rows, err := tx.Query(QueryGetAttributesByConsentID.Query, consentID, orgID)
    if err != nil {
        return nil, err
    }
    
    attributes := make([]model.ConsentAttribute, 0, len(rows))
    for _, row := range rows {
        attr := mapToConsentAttribute(row)
        if attr != nil {
            attributes = append(attributes, *attr)
        }
    }
    
    return attributes, nil
}

// DeleteAttributesByConsentID deletes all attributes within a transaction
func (s *store) DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
    _, err := tx.Exec(QueryDeleteAttributesByConsentID.Query, consentID, orgID)
    return err
}

// GetStatusAuditByConsentID retrieves status audit history within a transaction
func (s *store) GetStatusAuditByConsentID(tx dbmodel.TxInterface, consentID, orgID string) ([]model.ConsentStatusAudit, error) {
    rows, err := tx.Query(QueryGetStatusAuditByConsentID.Query, consentID, orgID)
    if err != nil {
        return nil, err
    }
    
    audits := make([]model.ConsentStatusAudit, 0, len(rows))
    for _, row := range rows {
        audit := mapToStatusAudit(row)
        if audit != nil {
            audits = append(audits, *audit)
        }
    }
    
    return audits, nil
}
```

**Delete old WithTx methods (lines 383-407):**
- Remove `CreateWithTx`
- Remove `CreateAttributesWithTx`
- Remove `CreateStatusAuditWithTx`

**Add executeTransaction helper (add at end of file):**

```go
// executeTransaction executes multiple queries within a single transaction
// This follows Thunder's functional composition pattern
func executeTransaction(dbClient provider.DBClientInterface, queries []func(tx dbmodel.TxInterface) error) error {
    tx, err := dbClient.BeginTx()
    if err != nil {
        return err
    }
    
    for _, query := range queries {
        if err := query(tx); err != nil {
            tx.Rollback()
            return err
        }
    }
    
    return tx.Commit()
}
```

**Lines removed:** ~200  
**Lines added:** ~150  
**Net change:** -50 lines, much cleaner

---

### File 2: `/consent-server/internal/authresource/store.go`

#### Changes Summary
- Same pattern as consent store
- Interface methods accept `tx dbmodel.TxInterface`
- Remove duplicate methods
- `executeTransaction` helper already exists (line 237) ✅

#### Interface Changes

**BEFORE (lines 69-84):**
```go
type authResourceStore interface {
    Create(ctx context.Context, authResource *model.AuthResource) error
    GetByID(ctx context.Context, authID, orgID string) (*model.AuthResource, error)
    GetByConsentID(ctx context.Context, consentID, orgID string) ([]model.AuthResource, error)
    Update(ctx context.Context, authResource *model.AuthResource) error
    UpdateStatus(ctx context.Context, authID, orgID, status string, updatedTime int64) error
    Delete(ctx context.Context, authID, orgID string) error
    DeleteByConsentID(ctx context.Context, consentID, orgID string) error
    Exists(ctx context.Context, authID, orgID string) (bool, error)
    GetByUserID(ctx context.Context, userID, orgID string) ([]model.AuthResource, error)
    UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID, status string, updatedTime int64) error
    
    // Duplicate transactional methods ❌
    CreateWithTx(ctx context.Context, tx *database.Tx, authResource *model.AuthResource) error
    UpdateAllStatusByConsentIDWithTx(ctx context.Context, tx *database.Tx, consentID, orgID, status string, updatedTime int64) error
}
```

**AFTER:**
```go
type authResourceStore interface {
    Create(tx dbmodel.TxInterface, authResource *model.AuthResource) error
    GetByID(tx dbmodel.TxInterface, authID, orgID string) (*model.AuthResource, error)
    GetByConsentID(tx dbmodel.TxInterface, consentID, orgID string) ([]model.AuthResource, error)
    Update(tx dbmodel.TxInterface, authResource *model.AuthResource) error
    UpdateStatus(tx dbmodel.TxInterface, authID, orgID, status string, updatedTime int64) error
    Delete(tx dbmodel.TxInterface, authID, orgID string) error
    DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
    Exists(tx dbmodel.TxInterface, authID, orgID string) (bool, error)
    GetByUserID(tx dbmodel.TxInterface, userID, orgID string) ([]model.AuthResource, error)
    UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
}
```

#### Implementation Pattern

Same as consent store - all methods use `tx.Exec()` or `tx.Query()` directly.

**Note:** `executeTransaction` helper already exists at line 237, keep it! ✅

**Lines removed:** ~150  
**Lines added:** ~100  
**Net change:** -50 lines

---

### File 3: `/consent-server/internal/consentpurpose/store.go`

#### Changes Summary
- Same pattern as other stores
- All methods accept `tx dbmodel.TxInterface`
- Add `executeTransaction` helper

#### Interface Changes

**BEFORE:**
```go
type consentPurposeStore interface {
    Create(ctx context.Context, purpose *model.ConsentPurpose) error
    GetByID(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, error)
    GetByName(ctx context.Context, name, orgID string) (*model.ConsentPurpose, error)
    List(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentPurpose, int, error)
    Update(ctx context.Context, purpose *model.ConsentPurpose) error
    Delete(ctx context.Context, purposeID, orgID string) error
    CheckNameExists(ctx context.Context, name, orgID string) (bool, error)
    CreateAttributes(ctx context.Context, attributes []model.ConsentPurposeAttribute) error
    GetAttributesByPurposeID(ctx context.Context, purposeID, orgID string) ([]model.ConsentPurposeAttribute, error)
    DeleteAttributesByPurposeID(ctx context.Context, purposeID, orgID string) error
    
    // Duplicate transactional method ❌
    LinkPurposeToConsentWithTx(ctx context.Context, tx *database.Tx, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}
```

**AFTER:**
```go
type consentPurposeStore interface {
    Create(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
    GetByID(tx dbmodel.TxInterface, purposeID, orgID string) (*model.ConsentPurpose, error)
    GetByName(tx dbmodel.TxInterface, name, orgID string) (*model.ConsentPurpose, error)
    List(tx dbmodel.TxInterface, orgID string, limit, offset int) ([]model.ConsentPurpose, int, error)
    Update(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
    Delete(tx dbmodel.TxInterface, purposeID, orgID string) error
    CheckNameExists(tx dbmodel.TxInterface, name, orgID string) (bool, error)
    CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentPurposeAttribute) error
    GetAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) ([]model.ConsentPurposeAttribute, error)
    DeleteAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) error
    LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}
```

**Add executeTransaction helper** (same as consent store)

**Lines removed:** ~150  
**Lines added:** ~100  
**Net change:** -50 lines

---

### File 4: `/consent-server/internal/consent/service.go`

#### Changes Summary
- Update exported interfaces (simpler method names)
- Replace `db *database.DB` with `dbClient provider.DBClientInterface`
- Refactor all service methods to use `executeTransaction`
- No manual tx management

#### Exported Interface Changes

**BEFORE (lines 28-38):**
```go
type AuthResourceStore interface {
    CreateWithTx(ctx context.Context, tx *database.Tx, authResource *authmodel.AuthResource) error
    UpdateAllStatusByConsentIDWithTx(ctx context.Context, tx *database.Tx, consentID, orgID, status string, updatedTime int64) error
}

type ConsentPurposeStore interface {
    LinkPurposeToConsentWithTx(ctx context.Context, tx *database.Tx, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}
```

**AFTER:**
```go
type AuthResourceStore interface {
    Create(tx dbmodel.TxInterface, authResource *authmodel.AuthResource) error
    UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
}

type ConsentPurposeStore interface {
    LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}
```

#### Service Struct Changes

**BEFORE:**
```go
type consentService struct {
    store               consentStore
    authResourceStore   AuthResourceStore
    consentPurposeStore ConsentPurposeStore
    db                  *database.DB  // ❌ Remove
}

func newConsentService(store consentStore, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore, db *database.DB) ConsentService {
    return &consentService{
        store:               store,
        authResourceStore:   authResourceStore,
        consentPurposeStore: consentPurposeStore,
        db:                  db,
    }
}
```

**AFTER:**
```go
type consentService struct {
    store               consentStore
    authResourceStore   AuthResourceStore
    consentPurposeStore ConsentPurposeStore
    dbClient            provider.DBClientInterface  // ✅ Add
}

func newConsentService(store consentStore, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore, dbClient provider.DBClientInterface) ConsentService {
    return &consentService{
        store:               store,
        authResourceStore:   authResourceStore,
        consentPurposeStore: consentPurposeStore,
        dbClient:            dbClient,
    }
}
```

#### CreateConsent Refactor

**BEFORE (lines 58-206): Manual tx management**
```go
func (s *consentService) CreateConsent(...) (*model.ConsentResponse, *serviceerror.ServiceError) {
    // Validate...
    
    tx, err := s.db.BeginTx(ctx)  // ❌ Manual tx
    if err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to begin transaction: %v", err))
    }
    defer tx.Rollback()  // ❌ Manual rollback
    
    // Create consent
    if err := s.store.CreateWithTx(ctx, tx, consent); err != nil {  // ❌ WithTx methods
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
    }
    
    // Create attributes
    if err := s.store.CreateAttributesWithTx(ctx, tx, attributes); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create attributes: %v", err))
    }
    
    // Create audit
    if err := s.store.CreateStatusAuditWithTx(ctx, tx, audit); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create status audit: %v", err))
    }
    
    // Create auth resources
    for _, authReq := range req.Authorizations {
        if err := s.authResourceStore.CreateWithTx(ctx, tx, authResource); err != nil {
            return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create auth resource: %v", err))
        }
    }
    
    if err := tx.Commit(); err != nil {  // ❌ Manual commit
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to commit transaction: %v", err))
    }
    
    return response, nil
}
```

**AFTER: Functional composition with executeTransaction**
```go
func (s *consentService) CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
    // Validate
    if err := utils.ValidateOrgID(orgID); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
    }
    if err := utils.ValidateClientID(clientID); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
    }
    if err := validator.ValidateConsentCreateRequest(req, clientID, orgID); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
    }
    
    // Convert and prepare data
    createReq, err := req.ToConsentCreateRequest()
    if err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
    }
    
    consentID := utils.GenerateUUID()
    currentTime := utils.GetCurrentTimestamp()
    
    consent := &model.Consent{
        ConsentID:                   consentID,
        CreatedTime:                 currentTime,
        UpdatedTime:                 currentTime,
        ClientID:                    clientID,
        ConsentType:                 createReq.ConsentType,
        CurrentStatus:               createReq.CurrentStatus,
        ConsentFrequency:            createReq.ConsentFrequency,
        ValidityTime:                createReq.ValidityTime,
        RecurringIndicator:          createReq.RecurringIndicator,
        DataAccessValidityDuration: createReq.DataAccessValidityDuration,
        OrgID:                       orgID,
    }
    
    // Build queries array ✅
    queries := []func(tx dbmodel.TxInterface) error{
        // Create consent
        func(tx dbmodel.TxInterface) error {
            return s.store.Create(tx, consent)
        },
    }
    
    // Add attributes if provided
    if len(createReq.Attributes) > 0 {
        attributes := make([]model.ConsentAttribute, 0, len(createReq.Attributes))
        for key, value := range createReq.Attributes {
            attributes = append(attributes, model.ConsentAttribute{
                ConsentID: consentID,
                AttKey:    key,
                AttValue:  value,
                OrgID:     orgID,
            })
        }
        queries = append(queries, func(tx dbmodel.TxInterface) error {
            return s.store.CreateAttributes(tx, attributes)
        })
    }
    
    // Add status audit
    auditID := utils.GenerateUUID()
    audit := &model.ConsentStatusAudit{
        StatusAuditID: auditID,
        ConsentID:     consentID,
        CurrentStatus: createReq.CurrentStatus,
        ActionTime:    currentTime,
        OrgID:         orgID,
    }
    queries = append(queries, func(tx dbmodel.TxInterface) error {
        return s.store.CreateStatusAudit(tx, audit)
    })
    
    // Add auth resources
    for _, authReq := range req.Authorizations {
        authID := utils.GenerateUUID()
        var resourcesJSON *string
        if authReq.Resources != nil {
            resourcesBytes, err := json.Marshal(authReq.Resources)
            if err != nil {
                return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal resources: %v", err))
            }
            resourcesStr := string(resourcesBytes)
            resourcesJSON = &resourcesStr
        }
        
        authResource := &authmodel.AuthResource{
            AuthID:      authID,
            ConsentID:   consentID,
            AuthType:    authReq.AuthType,
            UserID:      authReq.UserID,
            AuthStatus:  authReq.Status,
            UpdatedTime: currentTime,
            Resources:   resourcesJSON,
            OrgID:       orgID,
        }
        
        queries = append(queries, func(tx dbmodel.TxInterface) error {
            return s.authResourceStore.Create(tx, authResource)
        })
    }
    
    // Execute all queries in single transaction ✅
    if err := executeTransaction(s.dbClient, queries); err != nil {
        return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
    }
    
    // Build response
    return &model.ConsentResponse{
        ConsentID:                   consent.ConsentID,
        CreatedTime:                 consent.CreatedTime,
        UpdatedTime:                 consent.UpdatedTime,
        ClientID:                    consent.ClientID,
        ConsentType:                 consent.ConsentType,
        CurrentStatus:               consent.CurrentStatus,
        ConsentFrequency:            consent.ConsentFrequency,
        ValidityTime:                consent.ValidityTime,
        RecurringIndicator:          consent.RecurringIndicator,
        DataAccessValidityDuration: consent.DataAccessValidityDuration,
        OrgID:                       consent.OrgID,
    }, nil
}
```

#### Other Service Methods

Apply same pattern to:
- `GetConsent` - use `executeTransaction` with read queries
- `ListConsents` - use `executeTransaction` with list queries
- `UpdateConsent` - use `executeTransaction` with update queries
- `UpdateConsentStatus` - use `executeTransaction` with status update + audit
- `DeleteConsent` - use `executeTransaction` with delete queries

**Lines removed:** ~100  
**Lines added:** ~120 (more explicit but cleaner)  
**Net change:** +20 lines but much more maintainable

---

### File 5: `/consent-server/internal/consent/init.go`

#### Changes

**BEFORE (line 14):**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface, db *database.DB, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore) ConsentService {
    store := newConsentStore(dbClient)
    service := newConsentService(store, authResourceStore, consentPurposeStore, db)
    handler := newConsentHandler(service)
    registerRoutes(mux, handler)
    return service
}
```

**AFTER:**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore) ConsentService {
    store := newConsentStore(dbClient)
    service := newConsentService(store, authResourceStore, consentPurposeStore, dbClient)  // ✅ Pass dbClient instead of db
    handler := newConsentHandler(service)
    registerRoutes(mux, handler)
    return service
}
```

**Lines changed:** 2

---

### File 6: `/consent-server/internal/authresource/init.go`

#### Changes

**BEFORE:**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) AuthResourceServiceInterface {
    store := newAuthResourceStore(dbClient)
    service := newAuthResourceService(store)
    handler := newAuthResourceHandler(service)
    registerRoutes(mux, handler)
    return service
}
```

**AFTER:**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) (AuthResourceServiceInterface, authResourceStore) {
    store := newAuthResourceStore(dbClient)
    service := newAuthResourceService(store)
    handler := newAuthResourceHandler(service)
    registerRoutes(mux, handler)
    return service, store  // ✅ Return both service and store
}
```

**Lines changed:** 1

---

### File 7: `/consent-server/internal/consentpurpose/init.go`

#### Changes

**BEFORE:**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) ConsentPurposeService {
    store := newConsentPurposeStore(dbClient)
    service := newConsentPurposeService(store)
    handler := newConsentPurposeHandler(service)
    registerRoutes(mux, handler)
    return service
}
```

**AFTER:**
```go
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) (ConsentPurposeService, consentPurposeStore) {
    store := newConsentPurposeStore(dbClient)
    service := newConsentPurposeService(store)
    handler := newConsentPurposeHandler(service)
    registerRoutes(mux, handler)
    return service, store  // ✅ Return both service and store
}
```

**Lines changed:** 1

---

### File 8: `/consent-server/cmd/server/servicemanager.go`

#### Changes

**BEFORE (line 22-46):**
```go
func registerServices(
    mux *http.ServeMux,
    dbClient provider.DBClientInterface,
    db *database.DB,  // ❌ Remove this parameter
) {
    logger := log.GetLogger()
    
    var authStore consent.AuthResourceStore
    authResourceService, authStore = authresource.Initialize(mux, dbClient)
    logger.Info("AuthResource module initialized")
    
    var purposeStore consent.ConsentPurposeStore
    consentPurposeService, purposeStore = consentpurpose.Initialize(mux, dbClient)
    logger.Info("ConsentPurpose module initialized")
    
    consentService = consent.Initialize(mux, dbClient, db, authStore, purposeStore)  // ❌ Remove db param
    logger.Info("Consent module initialized")
    
    // Health check...
}
```

**AFTER:**
```go
func registerServices(
    mux *http.ServeMux,
    dbClient provider.DBClientInterface,  // ✅ Only dbClient needed
) {
    logger := log.GetLogger()
    
    var authStore consent.AuthResourceStore
    authResourceService, authStore = authresource.Initialize(mux, dbClient)
    logger.Info("AuthResource module initialized")
    
    var purposeStore consent.ConsentPurposeStore
    consentPurposeService, purposeStore = consentpurpose.Initialize(mux, dbClient)
    logger.Info("ConsentPurpose module initialized")
    
    consentService = consent.Initialize(mux, dbClient, authStore, purposeStore)  // ✅ No db param
    logger.Info("Consent module initialized")
    
    // Health check...
}
```

**Lines changed:** 2

---

### File 9: `/consent-server/cmd/server/main.go`

#### Changes

**BEFORE:**
```go
registerServices(mux, dbClient, db)
```

**AFTER:**
```go
registerServices(mux, dbClient)
```

**Lines changed:** 1

---

## Summary

### Files Modified: 9

| File | Lines Removed | Lines Added | Net Change |
|------|---------------|-------------|------------|
| `consent/store.go` | ~200 | ~150 | **-50** |
| `authresource/store.go` | ~150 | ~100 | **-50** |
| `consentpurpose/store.go` | ~150 | ~100 | **-50** |
| `consent/service.go` | ~100 | ~120 | **+20** |
| `consent/init.go` | 1 | 1 | **0** |
| `authresource/init.go` | 1 | 1 | **0** |
| `consentpurpose/init.go` | 1 | 1 | **0** |
| `cmd/server/servicemanager.go` | 2 | 1 | **-1** |
| `cmd/server/main.go` | 1 | 1 | **0** |
| **TOTAL** | **~606** | **~475** | **-131** |

### Benefits

✅ **No interface duplication** - Single `Create()` method instead of `Create()` + `CreateWithTx()`  
✅ **Consistent transaction pattern** - All operations use `executeTransaction`  
✅ **Thunder-aligned** - Functional composition with query builders  
✅ **Easier to test** - Query functions are easily mockable  
✅ **Cleaner code** - No manual tx management, automatic rollback  
✅ **Flexible** - Easy to compose queries from multiple stores  
✅ **ACID guarantees** - Cross-module operations in single transaction  

### Testing Checklist

After implementation, verify:
- [ ] `go build` succeeds
- [ ] Create consent creates all related entities atomically
- [ ] Transaction rollback works on errors
- [ ] Read operations work correctly
- [ ] Update operations work correctly
- [ ] Delete operations clean up all related data
- [ ] Cross-module operations maintain data consistency
- [ ] All existing API endpoints still function

---

## Next Steps

1. **Review this plan** - Confirm approach is correct
2. **Implement changes** - Apply refactoring to all 9 files
3. **Build and test** - Verify `go build` succeeds
4. **Test transactions** - Verify atomic operations and rollback
5. **Update documentation** - Document the new transaction pattern

---

*Generated: 10 December 2025*
