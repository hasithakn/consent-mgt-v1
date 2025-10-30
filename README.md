# Consent Management API

A comprehensive RESTful API service for managing consent lifecycle including initiation, authorization, validation, administration, and revocation.

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│         API Handlers (HTTP Layer)       │
│  POST/GET/PUT /consents, /auth, /file   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│       Core Services (Business Logic)    │
│  ConsentService, AuthService, FileServ  │
│  + Extension Point Client Integration   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│          DAO Layer (Data Access)        │
│  ConsentDAO, AuthDAO, FileDAO, etc.     │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│       Database (MySQL with sqlx)        │
│  FS_CONSENT, FS_CONSENT_AUTH_RESOURCE   │
└─────────────────────────────────────────┘

        ┌────────────────────┐
        │ Extension Service  │ ← HTTP Client calls
        │   (External API)   │
        └────────────────────┘
```

## 📁 Project Structure

```
consent-mgt-v1/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go
├── internal/
│   ├── models/              # Database models (structs)
│   ├── config/              # Configuration management
│   ├── database/            # Database connection & transactions
│   ├── dao/                 # Data Access Objects
│   ├── service/             # Business logic layer
│   ├── handlers/            # HTTP request handlers
│   ├── middleware/          # HTTP middleware (auth, logging, etc.)
│   ├── router/              # Route definitions
│   └── client/              # External service clients
├── pkg/
│   └── utils/               # Shared utilities
├── configs/                 # Configuration files
│   ├── config.yaml
│   ├── config.dev.yaml
│   └── config.prod.yaml
├── migrations/              # Database migration scripts
│   └── db_schema_mysql.sql
├── tests/
│   ├── dao/                 # DAO unit tests
│   ├── service/             # Service unit tests
│   ├── handlers/            # Handler unit tests
│   └── integration/         # Integration tests
├── docs/                    # Documentation
│   └── API.md
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## 🚀 Features

- **Consent Lifecycle Management**: Create, retrieve, update, and revoke consents
- **Authorization Management**: Handle authorization resources for consents
- **File Management**: Upload, download, and update consent-related files
- **Audit Trail**: Complete status change tracking
- **Multi-tenancy**: Organization-level data isolation
- **Extension Points**: Customizable hooks for pre/post processing
- **RESTful API**: Following OpenAPI 3.0 specification
- **Database Support**: MySQL with connection pooling

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: MySQL
- **Database Layer**: sqlx
- **Configuration**: Viper
- **Logging**: Logrus
- **Testing**: Testify, sqlmock, testcontainers
- **UUID**: Google UUID

## 📋 Prerequisites

- Go 1.21 or higher
- MySQL 8.0 or higher
- Docker & Docker Compose (optional)

## ⚙️ Installation & Setup

### 1. Clone the repository

```bash
git clone <repository-url>
cd consent-mgt-v1
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Setup Database

```bash
# Create database (if not exists)
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS AATest;"

# Run migrations
cd migrations
chmod +x migrate.sh
./migrate.sh
cd ..
```

### 4. Configure the application

Edit `configs/config.yaml` to match your environment:

```yaml
server:
  port: 9446
  host: localhost

database:
  host: localhost
  port: 3306
  user: root
  password: root
  database: AATest
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetimeMinutes: 5

logging:
  level: info  # debug, info, warn, error
  format: json
```

## 🚀 Running the Server

### Quick Start

```bash
# Run the server
go run cmd/server/main.go

# Server starts at http://localhost:3000
# Health check: curl http://localhost:3000/health
```

### Build & Run

```bash
# Build binary
go build -o consent-api-server cmd/server/main.go

# Run binary
./consent-api-server

# With custom config
CONFIG_PATH=configs/config.yaml ./consent-api-server
```

### Stop Server

Press `Ctrl+C` to gracefully shutdown

## 🐳 Docker Deployment

```bash
# Build and start services
docker-compose up --build

# Stop services
docker-compose down
```

## 🧪 Testing

### Run All Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...
```

### Run Integration Tests

```bash
# Run integration tests
go test ./tests/integration/... -v

# Run specific test
go test ./tests/integration/... -run TestCreateConsent -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Requirements

- MySQL database running on `localhost:3306`
- Database: `AAconsent-mgt-v3` (configured in test setup)
- User: `root` / Password: `password`

### Manual API Testing

Once the server is running, you can test the API endpoints:

#### Create a Consent

```bash
curl -X POST http://localhost:9446/api/v1/consents \
  -H "Content-Type: application/json" \
  -H "org-id: TEST_ORG" \
  -H "client-id: test-client-123" \
  -d '{
    "receipt": {
      "data": "Sample consent receipt",
      "purpose": "Account access"
    },
    "consentType": "accounts",
    "currentStatus": "awaitingAuthorization",
    "validityTime": 7776000,
    "recurringIndicator": false,
    "attributes": {
      "source": "mobile-app"
    }
  }'
```

#### Create Consent with Authorization Resources

```bash
curl -X POST http://localhost:9446/api/v1/consents \
  -H "Content-Type: application/json" \
  -H "org-id: TEST_ORG" \
  -H "client-id: test-client-456" \
  -d '{
    "receipt": {
      "data": "Consent with auth"
    },
    "consentType": "payments",
    "currentStatus": "awaitingAuthorization",
    "validityTime": 2592000,
    "authResources": [
      {
        "authType": "authorization_code",
        "authStatus": "authorized",
        "userId": "user-123",
        "resource": {
          "scopes": ["read", "write"]
        }
      }
    ]
  }'
```

### Test Results

Current test status: **30/30 tests passing ✓**

- ✅ 10 Authorization Resource Integration Tests
- ✅ 17 Consent Service Integration Tests
- ✅ 3 API Integration Tests

## 📚 API Documentation

The API follows RESTful principles and is based on the OpenAPI 3.0 specification.

### Base URL

```
http://localhost:9446/api/v1
```

### Implemented Endpoints

#### ✅ Consents

- `POST /consents` - Create a new consent (with optional auth resources)
  - **Status**: ✅ Implemented & Tested
  - **Headers**: `org-id`, `client-id`

### Planned Endpoints

#### Consents (Remaining)

- `GET /consents/{consentId}` - Retrieve consent by ID
- `PUT /consents/{consentId}` - Update consent
- `GET /consents` - Search consents with filters
- `POST /consents/{consentId}/revoke` - Revoke a consent

#### Authorization Resources

- `POST /consents/{consentId}/authorizations` - Create authorization resource
- `GET /consents/{consentId}/authorizations` - List authorization resources
- `GET /consents/{consentId}/authorizations/{authId}` - Get authorization resource
- `PUT /consents/{consentId}/authorizations/{authId}` - Update authorization resource
- `DELETE /consents/{consentId}/authorizations/{authId}` - Delete authorization resource

### Common Headers

All endpoints require the following headers:

- `org-id`: Organization identifier (for multi-tenancy)
- `client-id`: Client application identifier (optional, defaults from context)
- `Content-Type`: application/json (for POST/PUT requests)

## 🔧 Configuration

Configuration is managed through YAML files in the `configs/` directory:

```yaml
server:
  port: 9446
  host: localhost

database:
  host: localhost
  port: 3306
  user: root
  password: password
  database: consent_mgt
  maxOpenConns: 25
  maxIdleConns: 5

extension:
  baseUrl: https://extension-service:8080/api/services
  timeout: 30s

logging:
  level: info
  format: json
```

## 🔐 Security

- Basic Authentication for API endpoints
- Multi-tenancy with ORG_ID isolation
- SQL injection prevention through parameterized queries
- Input validation at handler level

## 📈 Development Status

This project is currently under development. See the TODO list for progress tracking.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

Apache 2.0 - See LICENSE file for details

## 📞 Contact

WSO2 - architecture@wso2.com

Project Link: [https://github.com/wso2/consent-management-api](https://github.com/wso2/consent-management-api)

---

**Note**: This README will be updated as the project progresses through development phases.
