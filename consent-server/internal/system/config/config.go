package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabasesConfig        `mapstructure:"database"`
	ServiceExtension ServiceExtensionConfig `mapstructure:"service_extension"`
	Logging          LoggingConfig          `mapstructure:"logging"`
	Consent          ConsentConfig          `mapstructure:"consent"`
	Security         SecurityConfig         `mapstructure:"security"`
	CORS             CORSConfig             `mapstructure:"cors"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Hostname     string        `mapstructure:"hostname"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
}

// DatabasesConfig holds all database configurations
type DatabasesConfig struct {
	Consent DatabaseConfig `mapstructure:"consent"`
}

// DatabaseConfig holds individual database configuration
type DatabaseConfig struct {
	Type            string        `mapstructure:"type"`
	Hostname        string        `mapstructure:"hostname"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// ServiceExtensionConfig holds extension service configuration
type ServiceExtensionConfig struct {
	Enabled       bool               `mapstructure:"enabled"`
	BaseURL       string             `mapstructure:"base_url"`
	Timeout       time.Duration      `mapstructure:"timeout"`
	RetryAttempts int                `mapstructure:"retry_attempts"`
	Endpoints     ExtensionEndpoints `mapstructure:"endpoints"`
}

// ExtensionEndpoints holds all extension service endpoint paths
type ExtensionEndpoints struct {
	PreProcessConsentCreation     string `mapstructure:"pre_process_consent_creation"`
	EnrichConsentCreationResponse string `mapstructure:"enrich_consent_creation_response"`
	PreProcessConsentRetrieval    string `mapstructure:"pre_process_consent_retrieval"`
	PreProcessConsentUpdate       string `mapstructure:"pre_process_consent_update"`
	EnrichConsentUpdateResponse   string `mapstructure:"enrich_consent_update_response"`
	PreProcessConsentRevoke       string `mapstructure:"pre_process_consent_revoke"`
	MapAcceleratorErrorResponse   string `mapstructure:"map_accelerator_error_response"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// ConsentStatus represents a typed consent status
type ConsentStatus string

// AuthStatus represents a typed authorization status
type AuthStatus string

// ConsentConfig holds consent-related configuration
type ConsentConfig struct {
	StatusMappings     ConsentStatusMappings `mapstructure:"status_mappings"`
	AuthStatusMappings AuthStatusMappings    `mapstructure:"auth_status_mappings"`
}

// ConsentStatusMappings holds the mapping of specific consent lifecycle states
type ConsentStatusMappings struct {
	ActiveStatus   string `mapstructure:"active_status"`
	ExpiredStatus  string `mapstructure:"expired_status"`
	RevokedStatus  string `mapstructure:"revoked_status"`
	CreatedStatus  string `mapstructure:"created_status"`
	RejectedStatus string `mapstructure:"rejected_status"`
}

// AuthStatusMappings holds the mapping of authorization resource lifecycle states
type AuthStatusMappings struct {
	ApprovedState      string `mapstructure:"approved_state"`
	RejectedState      string `mapstructure:"rejected_state"`
	CreatedState       string `mapstructure:"created_state"`
	SystemExpiredState string `mapstructure:"system_expired_state"`
	SystemRevokedState string `mapstructure:"system_revoked_state"`
}

// GetActiveConsentStatus returns the typed active status from config
func (c *ConsentConfig) GetActiveConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.ActiveStatus)
}

// GetExpiredConsentStatus returns the typed expired status from config
func (c *ConsentConfig) GetExpiredConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.ExpiredStatus)
}

// GetRevokedConsentStatus returns the typed revoked status from config
func (c *ConsentConfig) GetRevokedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.RevokedStatus)
}

// GetCreatedConsentStatus returns the typed created status from config
func (c *ConsentConfig) GetCreatedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.CreatedStatus)
}

// GetRejectedConsentStatus returns the typed rejected status from config
func (c *ConsentConfig) GetRejectedConsentStatus() ConsentStatus {
	return ConsentStatus(c.StatusMappings.RejectedStatus)
}

// GetApprovedAuthStatus returns the typed approved auth status from config
func (c *ConsentConfig) GetApprovedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.ApprovedState)
}

// GetRejectedAuthStatus returns the typed rejected auth status from config
func (c *ConsentConfig) GetRejectedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.RejectedState)
}

// GetCreatedAuthStatus returns the typed created auth status from config
func (c *ConsentConfig) GetCreatedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.CreatedState)
}

// GetSystemExpiredAuthStatus returns the typed system expired auth status from config
func (c *ConsentConfig) GetSystemExpiredAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.SystemExpiredState)
}

// GetSystemRevokedAuthStatus returns the typed system revoked auth status from config
func (c *ConsentConfig) GetSystemRevokedAuthStatus() AuthStatus {
	return AuthStatus(c.AuthStatusMappings.SystemRevokedState)
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	BasicAuth BasicAuthConfig `mapstructure:"basic_auth"`
}

// BasicAuthConfig holds basic authentication configuration
type BasicAuthConfig struct {
	Enabled bool            `mapstructure:"enabled"`
	Users   []BasicAuthUser `mapstructure:"users"`
}

// BasicAuthUser represents a basic auth user
type BasicAuthUser struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

var globalConfig *Config

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Default configuration lookup order:
		// 1. ./repository/conf/deployment.yaml (production - relative to binary)
		// 2. ./cmd/server/repository/conf/deployment.yaml (development)
		v.SetConfigName("deployment")
		v.SetConfigType("yaml")
		v.AddConfigPath("./repository/conf")            // Production: <binary_dir>/repository/conf/
		v.AddConfigPath("./cmd/server/repository/conf") // Development
		v.AddConfigPath("../repository/conf")           // If running from subdirectory
		v.AddConfigPath(".")
	}

	// Read from environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("CONSENT_MGT")

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	globalConfig = &config
	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Database.Consent.Hostname == "" {
		return fmt.Errorf("database hostname is required")
	}

	if config.Database.Consent.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if config.ServiceExtension.Enabled && config.ServiceExtension.BaseURL == "" {
		return fmt.Errorf("service extension base URL is required when extension is enabled")
	}

	// Validate consent status mappings
	if config.Consent.StatusMappings.ActiveStatus == "" {
		return fmt.Errorf("consent active status mapping is required")
	}
	if config.Consent.StatusMappings.ExpiredStatus == "" {
		return fmt.Errorf("consent expired status mapping is required")
	}
	if config.Consent.StatusMappings.RevokedStatus == "" {
		return fmt.Errorf("consent revoked status mapping is required")
	}
	if config.Consent.StatusMappings.CreatedStatus == "" {
		return fmt.Errorf("consent created status mapping is required")
	}
	if config.Consent.StatusMappings.RejectedStatus == "" {
		return fmt.Errorf("consent rejected status mapping is required")
	}

	// Validate auth status mappings
	if config.Consent.AuthStatusMappings.ApprovedState == "" {
		return fmt.Errorf("auth approved status mapping is required")
	}
	if config.Consent.AuthStatusMappings.RejectedState == "" {
		return fmt.Errorf("auth rejected status mapping is required")
	}
	if config.Consent.AuthStatusMappings.CreatedState == "" {
		return fmt.Errorf("auth created status mapping is required")
	}
	if config.Consent.AuthStatusMappings.SystemExpiredState == "" {
		return fmt.Errorf("auth system expired status mapping is required")
	}
	if config.Consent.AuthStatusMappings.SystemRevokedState == "" {
		return fmt.Errorf("auth system revoked status mapping is required")
	}

	return nil
}

// Get returns the global configuration
func Get() *Config {
	return globalConfig
}

// SetGlobal sets the global configuration (for testing purposes)
func SetGlobal(cfg *Config) {
	globalConfig = cfg
}

// GetDSN returns the database connection string
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		d.User,
		d.Password,
		d.Hostname,
		d.Port,
		d.Database,
	)
}

// GetServerAddress returns the server address in host:port format
func (s *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
}

// GetExtensionURL returns the full URL for a service extension endpoint
func (e *ServiceExtensionConfig) GetExtensionURL(endpoint string) string {
	return e.BaseURL + endpoint
}

// IsBasicAuthEnabled returns whether basic auth is enabled
func (s *SecurityConfig) IsBasicAuthEnabled() bool {
	return s.BasicAuth.Enabled
}

// ValidateUser validates basic auth credentials
func (s *SecurityConfig) ValidateUser(username, password string) bool {
	for _, user := range s.BasicAuth.Users {
		if user.Username == username && user.Password == password {
			return true
		}
	}
	return false
}

// IsStatusAllowed checks if a given status is a valid consent status
func (c *ConsentConfig) IsStatusAllowed(status ConsentStatus) bool {
	return status == c.GetActiveConsentStatus() ||
		status == c.GetExpiredConsentStatus() ||
		status == c.GetRevokedConsentStatus() ||
		status == c.GetCreatedConsentStatus() ||
		status == c.GetRejectedConsentStatus()
}

// IsActiveStatus checks if the given status represents an active consent
func (c *ConsentConfig) IsActiveStatus(status ConsentStatus) bool {
	return status == c.GetActiveConsentStatus()
}

// IsExpiredStatus checks if the given status represents an expired consent
func (c *ConsentConfig) IsExpiredStatus(status ConsentStatus) bool {
	return status == c.GetExpiredConsentStatus()
}

// IsRevokedStatus checks if the given status represents a revoked consent
func (c *ConsentConfig) IsRevokedStatus(status ConsentStatus) bool {
	return status == c.GetRevokedConsentStatus()
}

// IsCreatedStatus checks if the given status represents a created consent
func (c *ConsentConfig) IsCreatedStatus(status ConsentStatus) bool {
	return status == c.GetCreatedConsentStatus()
}

// IsRejectedStatus checks if the given status represents a rejected consent
func (c *ConsentConfig) IsRejectedStatus(status ConsentStatus) bool {
	return status == c.GetRejectedConsentStatus()
}

// IsTerminalStatus checks if the given status is a terminal state (expired or revoked)
func (c *ConsentConfig) IsTerminalStatus(status ConsentStatus) bool {
	return c.IsExpiredStatus(status) || c.IsRevokedStatus(status)
}

// GetAllowedConsentStatuses returns a list of all valid consent statuses
func (c *ConsentConfig) GetAllowedConsentStatuses() []ConsentStatus {
	return []ConsentStatus{
		c.GetCreatedConsentStatus(),
		c.GetActiveConsentStatus(),
		c.GetRejectedConsentStatus(),
		c.GetRevokedConsentStatus(),
		c.GetExpiredConsentStatus(),
	}
}

// IsAuthStatusAllowed checks if a given status is a valid authorization status
func (c *ConsentConfig) IsAuthStatusAllowed(status AuthStatus) bool {
	return status == c.GetCreatedAuthStatus() ||
		status == c.GetApprovedAuthStatus() ||
		status == c.GetRejectedAuthStatus() ||
		status == c.GetSystemExpiredAuthStatus() ||
		status == c.GetSystemRevokedAuthStatus()
}

// GetAllowedAuthStatuses returns a list of all valid authorization statuses
func (c *ConsentConfig) GetAllowedAuthStatuses() []AuthStatus {
	return []AuthStatus{
		c.GetCreatedAuthStatus(),
		c.GetApprovedAuthStatus(),
		c.GetRejectedAuthStatus(),
		c.GetSystemExpiredAuthStatus(),
		c.GetSystemRevokedAuthStatus(),
	}
}
