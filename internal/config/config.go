package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Extension ExtensionConfig `mapstructure:"extension"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Consent   ConsentConfig   `mapstructure:"consent"`
	Security  SecurityConfig  `mapstructure:"security"`
	CORS      CORSConfig      `mapstructure:"cors"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
}

// ExtensionConfig holds extension service configuration
type ExtensionConfig struct {
	Enabled       bool               `mapstructure:"enabled"`
	BaseURL       string             `mapstructure:"baseUrl"`
	Timeout       time.Duration      `mapstructure:"timeout"`
	RetryAttempts int                `mapstructure:"retryAttempts"`
	Endpoints     ExtensionEndpoints `mapstructure:"endpoints"`
}

// ExtensionEndpoints holds all extension service endpoint paths
type ExtensionEndpoints struct {
	PreProcessConsentCreation       string `mapstructure:"preProcessConsentCreation"`
	EnrichConsentCreationResponse   string `mapstructure:"enrichConsentCreationResponse"`
	PreProcessConsentRetrieval      string `mapstructure:"preProcessConsentRetrieval"`
	PreProcessConsentUpdate         string `mapstructure:"preProcessConsentUpdate"`
	EnrichConsentUpdateResponse     string `mapstructure:"enrichConsentUpdateResponse"`
	PreProcessConsentRevoke         string `mapstructure:"preProcessConsentRevoke"`
	PreProcessConsentFileUpload     string `mapstructure:"preProcessConsentFileUpload"`
	EnrichConsentFileResponse       string `mapstructure:"enrichConsentFileResponse"`
	ValidateConsentFileRetrieval    string `mapstructure:"validateConsentFileRetrieval"`
	PreProcessConsentFileUpdate     string `mapstructure:"preProcessConsentFileUpdate"`
	EnrichConsentFileUpdateResponse string `mapstructure:"enrichConsentFileUpdateResponse"`
	MapAcceleratorErrorResponse     string `mapstructure:"mapAcceleratorErrorResponse"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// ConsentConfig holds consent-related configuration
type ConsentConfig struct {
	AllowedStatuses []string              `mapstructure:"allowedStatuses"`
	StatusMappings  ConsentStatusMappings `mapstructure:"statusMappings"`
}

// ConsentStatusMappings holds the mapping of specific consent lifecycle states
type ConsentStatusMappings struct {
	ActiveStatus   string `mapstructure:"activeStatus"`
	ExpiredStatus  string `mapstructure:"expiredStatus"`
	RevokedStatus  string `mapstructure:"revokedStatus"`
	CreatedStatus  string `mapstructure:"createdStatus"`
	RejectedStatus string `mapstructure:"rejectedStatus"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	BasicAuth BasicAuthConfig `mapstructure:"basicAuth"`
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
	AllowedOrigins   []string `mapstructure:"allowedOrigins"`
	AllowedMethods   []string `mapstructure:"allowedMethods"`
	AllowedHeaders   []string `mapstructure:"allowedHeaders"`
	AllowCredentials bool     `mapstructure:"allowCredentials"`
	MaxAge           int      `mapstructure:"maxAge"`
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

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Extension.Enabled && config.Extension.BaseURL == "" {
		return fmt.Errorf("extension base URL is required when extension is enabled")
	}

	// Validate consent configuration
	if len(config.Consent.AllowedStatuses) == 0 {
		return fmt.Errorf("at least one allowed consent status is required")
	}

	if config.Consent.StatusMappings.ActiveStatus == "" {
		return fmt.Errorf("active status mapping is required")
	}

	if config.Consent.StatusMappings.ExpiredStatus == "" {
		return fmt.Errorf("expired status mapping is required")
	}

	if config.Consent.StatusMappings.RevokedStatus == "" {
		return fmt.Errorf("revoked status mapping is required")
	}

	// Verify that mapped statuses exist in allowed statuses
	mappedStatuses := []string{
		config.Consent.StatusMappings.ActiveStatus,
		config.Consent.StatusMappings.ExpiredStatus,
		config.Consent.StatusMappings.RevokedStatus,
		config.Consent.StatusMappings.CreatedStatus,
		config.Consent.StatusMappings.RejectedStatus,
	}

	allowedStatusMap := make(map[string]bool)
	for _, status := range config.Consent.AllowedStatuses {
		allowedStatusMap[status] = true
	}

	for _, mappedStatus := range mappedStatuses {
		if mappedStatus != "" && !allowedStatusMap[mappedStatus] {
			return fmt.Errorf("mapped status '%s' not found in allowedStatuses", mappedStatus)
		}
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
		d.Host,
		d.Port,
		d.Database,
	)
}

// GetServerAddress returns the server address in host:port format
func (s *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetExtensionURL returns the full URL for an extension endpoint
func (e *ExtensionConfig) GetExtensionURL(endpoint string) string {
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

// IsStatusAllowed checks if a given status is in the allowed statuses list
func (c *ConsentConfig) IsStatusAllowed(status string) bool {
	for _, allowedStatus := range c.AllowedStatuses {
		if allowedStatus == status {
			return true
		}
	}
	return false
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

// GetAllowedStatuses returns a copy of the allowed statuses list
func (c *ConsentConfig) GetAllowedStatuses() []string {
	statuses := make([]string, len(c.AllowedStatuses))
	copy(statuses, c.AllowedStatuses)
	return statuses
}
