# Integration Tests

This directory contains integration tests for the Consent Management API.

## Structure

```
integration-tests/
├── api/                           # REST API Integration Tests ✅
│   ├── auth_resource_api_test.go
│   ├── consent_api_test.go
│   ├── consent_purpose_api_test.go
│   ├── consent_attribute_search_test.go
│   ├── consent_revoke_test.go
│   └── consent_validate_test.go
├── dao-tests-archived/            # Archived DAO Tests (Not Run)
│   ├── auth_resource_integration_test.go
│   ├── consent_integration_test.go
│   └── consent_purpose_integration_test.go
├── go.mod                         # Separate Go module
├── go.sum
└── README.md                      # This file
```

## About

### Separate Go Module
The integration tests are in a **separate Go module** (`github.com/wso2/consent-management-api/tests`) that imports the main application as a dependency. This ensures:

- ✅ **Tests are not packaged** with the production binary
- ✅ **Faster builds** - test dependencies don't bloat the main module
- ✅ **Clear separation** between production code and tests

### API Tests (api/)
These tests verify the **REST API endpoints** by:
- Starting an HTTP test server
- Making HTTP requests
- Validating responses
- Testing against a real database

**What they test:**
- HTTP request/response handling
- Request validation
- Authentication/authorization
- Error responses
- API contracts

### DAO Tests (dao-tests-archived/)
These are **archived** tests that directly test the Data Access Layer. They've been moved here because:
- DAO layer is better tested through API tests
- Unit tests cover business logic
- Reduces test maintenance burden

## Running Tests

### Prerequisites
1. MySQL database running
2. Database configured in `../configs/config.yaml`

### Run All API Tests
```bash
cd integration-tests
go test ./api/... -v
```

### Run Specific Test File
```bash
cd integration-tests
go test ./api/consent_purpose_api_test.go -v
```

### Run Specific Test
```bash
cd integration-tests
go test ./api -run TestCreateConsentPurpose_Success -v
```

### With Coverage
```bash
cd integration-tests
go test ./api/... -cover
```

## Test Database Setup

The tests expect:
- MySQL database running on `localhost:3306`
- Test database: `consent_management_test` (or as configured)
- Proper schema created (run `../db_scripts/db_schema_mysql.sql`)

## Best Practices

1. **Cleanup**: Each test should clean up its test data
2. **Isolation**: Tests should not depend on each other
3. **Naming**: Use descriptive test names: `Test<Feature>_<Scenario>`
4. **Organization**: Group related tests in the same file

## Adding New Tests

When adding new API integration tests:

1. Create test file in `api/` directory: `<feature>_api_test.go`
2. Use package name: `package integration`
3. Follow existing test patterns
4. Add cleanup logic
5. Document test scenarios

Example:
```go
package integration

func TestNewFeature_Success(t *testing.T) {
    // Setup
    env := setupTestEnvironment(t)
    defer cleanup(t, env)
    
    // Execute
    resp := makeRequest(env.Router, request)
    
    // Assert
    assert.Equal(t, http.StatusOK, resp.Code)
}
```

## Notes

- These tests require a running database
- Tests may take longer than unit tests
- Run before merging to ensure API contracts are maintained
- CI/CD should run these tests with a test database
