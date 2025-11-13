# Consent Management API

A RESTful API service for managing consents and consent purposes with support for dynamic attribute validation through pluggable type handlers.

## Features

- **Consent Management**: Create, retrieve, update, revoke consents
- **Consent Purposes**: Manage consent purposes with type-based validation
  - String type: Simple string values (no mandatory attributes)
  - JSON Schema type: Schema-based validation (requires `validationSchema`)
  - Attribute type: Resource-based attributes (requires `resourcePath` and `jsonPath`)
- **Authorization Resources**: Handle consent authorization resources
- **Multi-tenancy**: Organization-level isolation with `org-id` header
- **Extensible Type Handlers**: Pluggable architecture for custom purpose types

## Technology Stack

- **Go** 1.21+
- **Web Framework**: Gin
- **Database**: MySQL 8.0+
- **Database Driver**: sqlx
- **Configuration**: Viper
- **Testing**: Testify

## Prerequisites

- Go 1.21 or higher
- MySQL 8.0 or higher

## Project Structure

```
consent-mgt-v1/
├── cmd/server/                    # Application entry point
│   └── main.go
├── internal/
│   ├── models/                    # Data models & DTOs
│   ├── dao/                       # Data Access Objects
│   ├── service/                   # Business logic
│   ├── handlers/                  # HTTP handlers
│   ├── router/                    # Route definitions
│   ├── purpose_type_handlers/     # Type handler registry
│   │   ├── string_handler.go     # String type handler
│   │   ├── json_schema_handler.go # JSON Schema type handler
│   │   ├── attribute_handler.go  # Attribute type handler
│   │   └── registry.go           # Handler registration
│   ├── database/                  # Database connection
│   ├── config/                    # Configuration
│   └── utils/                     # Utilities
├── configs/
│   └── config.yaml               # Configuration file
├── db_scripts/
│   └── db_schema_mysql.sql       # Database schema
├── integration-tests/            # Integration tests
│   └── api/
│       ├── consent-purpose/      # 58 tests (CRUD + type handlers)
│       ├── auth_resource_api_test.go
│       ├── consent_api_test.go
│       └── ...
└── README.md
```

## Quick Start

### 1. Setup Database

```bash
# Create database
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS AAconsent_mgt_v3;"

# Import schema
mysql -u root -p AAconsent_mgt_v3 < db_scripts/db_schema_mysql.sql
```

### 2. Configure Application

Edit `configs/config.yaml`:

```yaml
server:
  port: 3000

database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  database: AAconsent_mgt_v3
  maxOpenConns: 25
  maxIdleConns: 5
```

### 3. Run the Server

**Option A: Run directly**
```bash
go run cmd/server/main.go
```

**Option B: Build executable and run**
```bash
# Build executable
go build -o consent-api-server cmd/server/main.go

# Run with default config (configs/config.yaml)
./consent-api-server

# Run with custom config
./consent-api-server -config=/path/to/config.yaml
```

Server starts at `http://localhost:3000`

Health check: `curl http://localhost:3000/health`

## Testing

### Run All Integration Tests

```bash
cd integration-tests
go test ./... -v
```

### Run Specific Test Suite

```bash
# Consent Purpose tests (58 tests)
go test ./api/consent-purpose/... -v

# Consent API tests
go test ./api/ -run TestConsent -v
```

### Test Database Setup

Integration tests require:
- MySQL running on `localhost:3306`
- Database: `AAconsent-mgt-v3`
- User/Password configured in test files

### Test Coverage

**Consent Purpose Tests** (58 tests total):
- **CREATE**: 24 tests
  - General CRUD: 11 tests
  - Type handlers: 13 tests (string, json-schema, attribute)
- **READ**: 13 tests
  - General: 4 tests
  - Type handlers: 7 tests
  - List operations: 2 tests
- **UPDATE**: 13 tests
  - General: 4 tests
  - Type handlers: 9 tests
- **DELETE**: 6 tests
  - General: 2 tests
  - Type handlers: 4 tests
- **VALIDATE**: 5 tests
