# Migration Plan: Consent Purpose → Consent Element

**Date**: January 18, 2026  
**Objective**: Rename "consent-purpose" to "consent-element" throughout the codebase to align with API specification

---

## 📋 Overview

This migration involves renaming the core entity from "Consent Purpose" to "Consent Element" across:
- API routes: `/consent-purposes` → `/consent-elements`
- Database tables and columns
- Go package names and folder structure
- Type handlers terminology
- All code references and documentation

---

## 🗂️ Phase 1: Database Schema Migration

### Tables to Rename

| Current Name | New Name | Notes |
|-------------|----------|-------|
| `CONSENT_PURPOSE` | `CONSENT_ELEMENT` | Main entity table |
| `CONSENT_PURPOSE_ATTRIBUTE` | `CONSENT_ELEMENT_PROPERTY` | Element properties table (renamed from attributes) |
| `CONSENT_PURPOSE_GROUP_MAPPING` | `CONSENT_PURPOSE_ELEMENT_MAPPING` | Group-to-element mapping |
| `CONSENT_PURPOSE_APPROVAL` | `CONSENT_ELEMENT_APPROVAL` | Element approval tracking |

### Foreign Key Constraints to Update
- `FK_CONSENT_PURPOSE_ATTRIBUTE_PURPOSE` → `FK_CONSENT_ELEMENT_PROPERTY_ELEMENT`
- All references in `CONSENT_PURPOSE_GROUP_MAPPING`
- All references in `CONSENT_PURPOSE_APPROVAL`

### Migration Script Required
1. Create backup script: `db_schema_mysql_backup_v1.sql`
2. Create migration script: `migration_purpose_to_element.sql`
   - Rename tables with `ALTER TABLE ... RENAME TO ...`
   - Update foreign key constraints
   - Update indexes
3. Create rollback script: `rollback_purpose_to_element.sql`

**Files to Create/Modify**:
- `consent-server/dbscripts/migration_purpose_to_element.sql` (new)
- `consent-server/dbscripts/rollback_purpose_to_element.sql` (new)
- `consent-server/dbscripts/db_schema_mysql.sql` (update)

---

## 📁 Phase 2: Folder Structure & Package Renaming

### Directory Renaming

```
consent-server/internal/
├── consentpurpose/          → consentelement/
│   ├── validators/          → validators/
│   │   ├── handler.go       (rename types)
│   │   ├── string_handler.go
│   │   ├── json_schema_handler.go
│   │   ├── attribute_handler.go
│   │   └── registry.go
│   ├── model/
│   │   └── consent_purpose.go → consent_element.go
│   ├── handler.go
│   ├── init.go
│   ├── service.go
│   └── store.go
```

### Package Rename
- Package declaration: `package consentpurpose` → `package consentelement`
- Import paths: `github.com/wso2/consent-management-api/internal/consentpurpose` → `.../consentelement`

### Files Affected (Import Updates)
- `consent-server/cmd/server/main.go`
- `consent-server/cmd/server/servicemanager.go`
- `consent-server/internal/consent/model/consent.go`
- `consent-server/internal/consent/service.go`
- `consent-server/internal/consent/store.go`
- All test files

---

## 🔧 Phase 3: Type & Struct Renaming

### Core Types to Rename

| Current Name | New Name |
|-------------|----------|
| `ConsentPurpose` | `ConsentElement` |
| `ConsentPurposeCreateRequest` | `ConsentElementCreateRequest` |
| `ConsentPurposeUpdateRequest` | `ConsentElementUpdateRequest` |
| `ConsentPurposeResponse` | `ConsentElementResponse` |
| `ConsentPurposeListResponse` | `ConsentElementListResponse` |
| `ConsentPurposeAttribute` | `ConsentElementProperty` | Note: Changed to Property for consistency |
| `ConsentPurposeMapping` | `ConsentElementMapping` |
| `ConsentPurposeItem` | `ConsentElementItem` (used in consent payloads) |

### Handler Interface Types

| Current Name | New Name |
|-------------|----------|
| `PurposeTypeHandler` | `ElementTypeHandler` |
| `PurposePropertySpec` | `ElementPropertySpec` |
| `StringPurposeTypeHandler` | `StringElementTypeHandler` |
| `JsonSchemaPurposeTypeHandler` | `JsonPayloadElementTypeHandler` |
| `AttributePurposeTypeHandler` | `ResourceFieldElementTypeHandler` |
| `PurposeHandlerRegistry` | `ElementTypeHandlerRegistry` |

### Variable/Function Names to Update
- `consentPurpose` → `consentElement`
- `purposeID` → `elementID`
- `purposes` → `elements`
- `CreatePurpose()` → `CreateElement()`
- `GetPurpose()` → `GetElement()`
- `UpdatePurpose()` → `UpdateElement()`
- `DeletePurpose()` → `DeleteElement()`
- `ListPurposes()` → `ListElements()`
- `ValidatePurposeType()` → `ValidateElementType()`

### Comment Updates
All documentation strings referencing "purpose" should be updated to "element"

---

## 🌐 Phase 4: API Route & Handler Updates

### Route Changes (main.go / servicemanager.go)

```go
// Before
consentpurposeRouter := router.Group("/consent-purposes")
consentpurpose.RegisterRoutes(consentpurposeRouter, ...)

// After  
consentelementRouter := router.Group("/consent-elements")
consentelement.RegisterRoutes(consentelementRouter, ...)
```

### HTTP Handler Updates
- Update all route registrations in `handler.go`
- Update path parameters: `purposeId` → `elementId`
- Update operation IDs in comments

---

### Phase 5: Test Updates ⏸️ **DEFERRED**

_Tests will be updated in a follow-up task_

### Phase 6: Documentation Updates ⏸️ **DEFERRED**

_Documentation will be updated in a follow-up task_

---

## 🔍 Phase 7: Validation & Quality Assurance

### Pre-Migration Checklist
- [ ] Create database backup
- [ ] Run all existing tests and record baseline
- [ ] Document current API endpoints
- [ ] Create migration rollback plan

### Post-Migration Validation
- [ ] Run database migration script
- [ ] Verify all tables renamed correctly
- [ ] Verify all foreign keys intact
- [ ] Build Go code: `go build ./...`
- [ ] Basic API smoke tests
- [ ] Verify error messages are consistent

### API Endpoint Verification (Basic)
```bash
# Test element creation
POST /consent-elements

# Test element retrieval
GET /consent-elements/{elementId}

# Test element listing
GET /consent-elements

# Test element update
PUT /consent-elements/{elementId}

# Test element deletion
DELETE /consent-elements/{elementId}

# Test element validation
POST /consent-elements/validate
```

---

## 📊 Impact Analysis

### High Impact Areas
1. **Database Schema** - Requires migration script
2. **Package Structure** - All imports must be updated
3. **API Routes** - Breaking change for API consumers
4. **Integration Tests** - Extensive updates required

### Medium Impact Areas
1. **Type Definitions** - Many structs to rename
2. **Service Layer** - Function signatures change
3. **Store Layer** - SQL queries need updates

### Low Impact Areas
1. **Business Logic** - Mostly unchanged
2. **Validation Logic** - Already uses type handlers
3. **Error Handling** - Messages need minor updates

---

## ⚠️ Risk Mitigation

### Risks
1. **Breaking Changes**: API consumers using `/consent-purposes` will break
2. **Data Migration**: Database migration could fail or lose data
3. **Import Cycles**: Package renaming could create circular dependencies
4. **Test Coverage**: Tests may miss edge cases after rename

### Mitigation Strategies
1. **Versioning**: Consider API versioning (v1 vs v2)
2. **Migration Testing**: Test migration on copy of production data
3. **Incremental Rollout**: Deploy to staging environment first
4. **Backward Compatibility**: Optionally support old endpoints temporarily

---

## 🚀 Execution Plan

### Step-by-Step Execution

1. **Preparation** (1 hour)
   - Backup current codebase
   - Create feature branch: `feature/rename-purpose-to-element`
   - Backup database schema

2. **Database Migration** (2 hours)
   - Write migration scripts
   - Test on local database
   - Verify foreign keys and indexes

3. **Folder & Package Rename** (1 hour)
   - Rename `consentpurpose/` to `consentelement/`
   - Update package declarations
   - Fix all import statements

4. **Type Renaming** (3 hours)
   - Rename all structs and types
   - Update handler interfaces
   - Update validators

5. **Code Updates** (4 hours)
   - Update service layer
   - Update store layer  
   - Update handler layer
   - Update main.go and servicemanager.go

6. **Test Updates** (3 hours)
   - Rename test folders
   - Update test cases
   - Update test data

7. **Documentation** (1 hour)
   - Update README files
   - Update inline comments
   - Update API docs

8. **Testing & Validation** (2 hours)
   - Run build verification
   - Basic API testing
   - Integration smoke tests

9. **Review & Deployment** (1 hour)
   - Code review
   - Merge to main
   - Deploy to staging

**Total Estimated Time**: ~14 hours (Tests & Docs Deferred)

**Deferred Items**:
- Phase 5: Test Updates (defer to later)
- Phase 6: Documentation Updates (defer to later)

---

## 📋 Checklist

### Phase 1: Database
- [ ] Create migration script
- [ ] Create rollback script
- [ ] Test migration locally
- [ ] Update schema documentation

### Phase 2: Folder Structure
- [ ] Rename `consentpurpose/` folder
- [ ] Update all import statements
- [ ] Update package declarations

### Phase 3: Types & Structs
- [ ] Rename all core types
- [ ] Rename handler types
- [ ] Update method signatures

### Phase 4: API & Routes
- [ ] Update route registrations
- [ ] Update handler methods
- [ ] Update path parameters

### Phase 5: Tests ⏸️ **DEFERRED**
- [ ] Defer to follow-up task

### Phase 6: Documentation ⏸️ **DEFERRED**
- [ ] Defer to follow-up task

### Phase 7: Validation
- [ ] Build verification
- [ ] Unit test execution
- [ ] Integration test execution
- [ ] API endpoint testing

---

## 📚 Reference Files

### Key Files to Modify (Ordered by Priority)

1. **Database**:
   - `consent-server/dbscripts/db_schema_mysql.sql`
   - `consent-server/dbscripts/migration_purpose_to_element.sql` (new)

2. **Core Models**:
   - `consent-server/internal/consentpurpose/model/consent_purpose.go`

3. **Validators**:
   - `consent-server/internal/consentpurpose/validators/handler.go`
   - `consent-server/internal/consentpurpose/validators/*.go`

4. **Service/Store/Handler**:
   - `consent-server/internal/consentpurpose/service.go`
   - `consent-server/internal/consentpurpose/store.go`
   - `consent-server/internal/consentpurpose/handler.go`

5. **Main App**:
   - `consent-server/cmd/server/main.go`
   - `consent-server/cmd/server/servicemanager.go`

6. **Tests**:
   - `tests/integration/consentpurpose/*.go`

---

## 🎯 Success Criteria

- ✅ All code compiles without errors
- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Database migration successful
- ✅ API endpoints respond correctly
- ✅ No breaking changes for consent operations (only purpose/element endpoints)
- ✅ Documentation updated
- ✅ Zero data loss from migration

---

## 🔄 Rollback Plan

If issues arise:

1. **Code Rollback**: Revert Git branch
2. **Database Rollback**: Execute `rollback_purpose_to_element.sql`
3. **Verification**: Run tests on rolled-back state
4. **Analysis**: Identify root cause before retry

---

## 📞 Support & Questions

**Migration Owner**: Development Team  
**Review Required**: Senior Developer, DBA  
**Approval Required**: Tech Lead, Product Owner

---

**Status**: 📝 Planning Phase  
**Next Step**: Begin Phase 1 - Database Migration Script Creation
