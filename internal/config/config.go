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

// ConsentConfig holds consent-related configuration
type ConsentConfig struct {
	StatusMappings ConsentStatusMappings `mapstructure:"status_mappings"`
}

// ConsentStatusMappings holds the mapping of specific consent lifecycle states
type ConsentStatusMappings struct {
	ActiveStatus   string `mapstructure:"active_status"`
	ExpiredStatus  string `mapstructure:"expired_status"`
	RevokedStatus  string `mapstructure:"revoked_status"`
	CreatedStatus  string `mapstructure:"created_status"`
	RejectedStatus string `mapstructure:"rejected_status"`
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
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
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
		return fmt.Errorf("active status mapping is required")
	}

	if config.Consent.StatusMappings.ExpiredStatus == "" {
		return fmt.Errorf("expired status mapping is required")
	}

	if config.Consent.StatusMappings.RevokedStatus == "" {
		return fmt.Errorf("revoked status mapping is required")
	}

	if config.Consent.StatusMappings.CreatedStatus == "" {
		return fmt.Errorf("created status mapping is required")
	}

	if config.Consent.StatusMappings.RejectedStatus == "" {
		return fmt.Errorf("rejected status mapping is required")
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
func (c *ConsentConfig) IsStatusAllowed(status string) bool {
	// Check if the status matches any of the configured status mappings
	return status == c.StatusMappings.ActiveStatus ||
		status == c.StatusMappings.ExpiredStatus ||
		status == c.StatusMappings.RevokedStatus ||
		status == c.StatusMappings.CreatedStatus ||
		status == c.StatusMappings.RejectedStatus
}

// IsActiveStatus checks if the given status represents an active consent
func (c *ConsentConfig) IsActiveStatus(status string) bool {
	return status == c.StatusMappings.ActiveStatus
}

// IsExpiredStatus checks if the given status represents an expired consent
func (c *ConsentConfig) IsExpiredStatus(status string) bool {
	return status == c.StatusMappings.ExpiredStatus
}

// IsRevokedStatus checks if the given status represents a revoked consent
func (c *ConsentConfig) IsRevokedStatus(status string) bool {
	return status == c.StatusMappings.RevokedStatus
}

// IsCreatedStatus checks if the given status represents a created consent
func (c *ConsentConfig) IsCreatedStatus(status string) bool {
	return status == c.StatusMappings.CreatedStatus
}

// IsRejectedStatus checks if the given status represents a rejected consent
func (c *ConsentConfig) IsRejectedStatus(status string) bool {
	return status == c.StatusMappings.RejectedStatus
}

// IsTerminalStatus checks if the given status is a terminal state (expired or revoked)
func (c *ConsentConfig) IsTerminalStatus(status string) bool {
	return c.IsExpiredStatus(status) || c.IsRevokedStatus(status)
}

// GetAllowedStatuses returns a list of all configured consent statuses
func (c *ConsentConfig) GetAllowedStatuses() []string {
	return []string{
		c.StatusMappings.CreatedStatus,
		c.StatusMappings.ActiveStatus,
		c.StatusMappings.RejectedStatus,
		c.StatusMappings.RevokedStatus,
		c.StatusMappings.ExpiredStatus,
	}
}
