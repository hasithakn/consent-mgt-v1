package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/models"

	"github.com/sirupsen/logrus"
)

// ExtensionClient handles communication with the external extension service
type ExtensionClient struct {
	httpClient *http.Client
	config     *config.ExtensionConfig
	logger     *logrus.Logger
}

// ExtensionRequest represents the request payload sent to extension service
type ExtensionRequest struct {
	ConsentID   string                 `json:"consentId,omitempty"`
	ClientID    string                 `json:"clientId"`
	OrgID       string                 `json:"orgId"`
	ConsentType string                 `json:"consentType,omitempty"`
	Receipt     map[string]interface{} `json:"receipt,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExtensionResponse represents the response from extension service
type ExtensionResponse struct {
	Success      bool                   `json:"success"`
	Modified     bool                   `json:"modified,omitempty"`
	Receipt      map[string]interface{} `json:"receipt,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ErrorCode    string                 `json:"errorCode,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationResponse represents validation result from extension service
type ValidationResponse struct {
	Valid        bool     `json:"valid"`
	ErrorCode    string   `json:"errorCode,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// NewExtensionClient creates a new extension client instance
func NewExtensionClient(cfg *config.ExtensionConfig, logger *logrus.Logger) *ExtensionClient {
	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	return &ExtensionClient{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		config: cfg,
		logger: logger,
	}
}

// CallExtension makes an HTTP POST request to the extension service endpoint
func (c *ExtensionClient) CallExtension(ctx context.Context, endpoint string, request *ExtensionRequest) (*ExtensionResponse, error) {
	// Skip if extension service is not configured
	if c.config.BaseURL == "" {
		c.logger.Debug("Extension service not configured, skipping call")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	url := c.config.BaseURL + endpoint

	// Marshal request body
	jsonData, err := json.Marshal(request)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal extension request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create extension request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add correlation ID if available in context
	if correlationID, ok := ctx.Value("correlationID").(string); ok {
		req.Header.Set("X-Correlation-ID", correlationID)
	}

	// Log request
	c.logger.WithFields(logrus.Fields{
		"url":       url,
		"consentID": request.ConsentID,
		"clientID":  request.ClientID,
		"orgID":     request.OrgID,
	}).Debug("Calling extension service")

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.WithError(err).WithField("duration", duration).Error("Extension service call failed")
		return nil, fmt.Errorf("extension service call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read extension response")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log response
	c.logger.WithFields(logrus.Fields{
		"statusCode": resp.StatusCode,
		"duration":   duration,
		"url":        url,
	}).Debug("Extension service response received")

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"response":   string(body),
		}).Warn("Extension service returned non-success status")

		// Try to parse error response
		var errResp ExtensionResponse
		if err := json.Unmarshal(body, &errResp); err == nil && !errResp.Success {
			return &errResp, nil
		}

		return nil, fmt.Errorf("extension service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var extResponse ExtensionResponse
	if err := json.Unmarshal(body, &extResponse); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal extension response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &extResponse, nil
}

// CallValidation makes a validation call to the extension service
func (c *ExtensionClient) CallValidation(ctx context.Context, endpoint string, request *ExtensionRequest) (*ValidationResponse, error) {
	// Skip if extension service is not configured
	if c.config.BaseURL == "" {
		c.logger.Debug("Extension service not configured, skipping validation")
		return &ValidationResponse{Valid: true}, nil
	}

	url := c.config.BaseURL + endpoint

	// Marshal request body
	jsonData, err := json.Marshal(request)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal validation request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create validation request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add correlation ID if available in context
	if correlationID, ok := ctx.Value("correlationID").(string); ok {
		req.Header.Set("X-Correlation-ID", correlationID)
	}

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.WithError(err).WithField("duration", duration).Error("Validation service call failed")
		return nil, fmt.Errorf("validation service call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read validation response")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var valResponse ValidationResponse
	if err := json.Unmarshal(body, &valResponse); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal validation response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &valResponse, nil
}

// BuildExtensionRequest creates an ExtensionRequest from a Consent model
func BuildExtensionRequest(consent *models.Consent, additionalData map[string]interface{}) *ExtensionRequest {
	req := &ExtensionRequest{
		ConsentID:   consent.ConsentID,
		ClientID:    consent.ClientID,
		OrgID:       consent.OrgID,
		ConsentType: consent.ConsentType,
		Data:        additionalData,
	}

	// Convert ConsentPurposes JSON to map
	if consent.ConsentPurposes != nil {
		var receiptMap map[string]interface{}
		if err := json.Unmarshal(consent.ConsentPurposes, &receiptMap); err == nil {
			req.Receipt = receiptMap
		}
	}

	return req
}

// BuildExtensionRequestFromMap creates an ExtensionRequest from raw data
func BuildExtensionRequestFromMap(consentID, clientID, orgID, consentType string, receipt map[string]interface{}, data map[string]interface{}) *ExtensionRequest {
	return &ExtensionRequest{
		ConsentID:   consentID,
		ClientID:    clientID,
		OrgID:       orgID,
		ConsentType: consentType,
		Receipt:     receipt,
		Data:        data,
	}
}

// IsExtensionEnabled checks if the extension service is configured
func (c *ExtensionClient) IsExtensionEnabled() bool {
	return c.config.BaseURL != ""
}

// GetEndpoint returns the full URL for a given endpoint path
func (c *ExtensionClient) GetEndpoint(path string) string {
	return c.config.BaseURL + path
}

// Close closes the HTTP client connections
func (c *ExtensionClient) Close() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}
