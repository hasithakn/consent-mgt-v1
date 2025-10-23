package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/wso2/consent-management-api/internal/models"
)

// PreProcessConsentCreation calls the pre-process-consent-creation extension endpoint
// This is called before a consent is created to allow preprocessing and validation
func (c *ExtensionClient) PreProcessConsentCreation(
	ctx context.Context,
	request *models.ConsentCreateRequest,
	headers map[string]string,
) (*models.PreProcessConsentCreationResponse, error) {
	// Skip if extension service is not configured
	if !c.IsExtensionEnabled() {
		c.logger.Debug("Extension service not configured, skipping pre-process-consent-creation")
		return nil, nil
	}

	// Check if endpoint is configured
	endpoint := c.config.Endpoints.PreProcessConsentCreation
	if endpoint == "" {
		c.logger.Debug("PreProcessConsentCreation endpoint not configured, skipping")
		return nil, nil
	}

	// Build extension request
	extRequest := &models.PreProcessConsentCreationRequest{
		RequestID: uuid.New().String(),
		Data: models.PreProcessConsentCreationRequestData{
			ConsentInitiationData: request.ToConsentInitiationData(),
			RequestHeaders:        headers,
		},
	}

	// Marshal request body
	jsonData, err := json.Marshal(extRequest)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal pre-process-consent-creation request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"endpoint":  endpoint,
		"requestId": extRequest.RequestID,
	}).Debug("Calling pre-process-consent-creation extension")

	// Create HTTP request with context
	url := c.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create pre-process-consent-creation request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Make HTTP call
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to call pre-process-consent-creation extension")
		return nil, fmt.Errorf("failed to call extension: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read pre-process-consent-creation response")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("Pre-process-consent-creation extension returned non-200 status")
		return nil, fmt.Errorf("extension returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var extResponse models.PreProcessConsentCreationResponse
	if err := json.Unmarshal(body, &extResponse); err != nil {
		c.logger.WithError(err).WithField("body", string(body)).Error("Failed to unmarshal pre-process-consent-creation response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"responseId": extResponse.ResponseID,
		"status":     extResponse.Status,
	}).Debug("Pre-process-consent-creation extension call completed")

	// Check if extension returned an error
	if extResponse.Status == "ERROR" {
		c.logger.WithFields(map[string]interface{}{
			"errorCode": extResponse.ErrorCode,
			"errorData": extResponse.ErrorData,
		}).Warn("Pre-process-consent-creation extension returned ERROR status")
		// Return the response so caller can handle the error appropriately
		return &extResponse, nil
	}

	return &extResponse, nil
}

// PreProcessConsentUpdate calls the pre-process-consent-update extension endpoint
// This is called before a consent is updated to allow preprocessing and validation
func (c *ExtensionClient) PreProcessConsentUpdate(
	ctx context.Context,
	consentID string,
	request *models.ConsentUpdateRequest,
	headers map[string]string,
) (*models.PreProcessConsentUpdateResponse, error) {
	// Skip if extension service is not configured
	if !c.IsExtensionEnabled() {
		c.logger.Debug("Extension service not configured, skipping pre-process-consent-update")
		return nil, nil
	}

	// Check if endpoint is configured
	endpoint := c.config.Endpoints.PreProcessConsentUpdate
	if endpoint == "" {
		c.logger.Debug("PreProcessConsentUpdate endpoint not configured, skipping")
		return nil, nil
	}

	// Build extension request
	extRequest := &models.PreProcessConsentUpdateRequest{
		RequestID: uuid.New().String(),
		Data: models.PreProcessConsentUpdateRequestData{
			ConsentID:             consentID,
			ConsentInitiationData: request.ToConsentInitiationData(),
			RequestHeaders:        headers,
		},
	}

	// Marshal request body
	jsonData, err := json.Marshal(extRequest)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal pre-process-consent-update request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"endpoint":  endpoint,
		"requestId": extRequest.RequestID,
		"consentId": consentID,
	}).Debug("Calling pre-process-consent-update extension")

	// Create HTTP request with context
	url := c.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create pre-process-consent-update request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Make HTTP call
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to call pre-process-consent-update extension")
		return nil, fmt.Errorf("failed to call extension: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read pre-process-consent-update response")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("Pre-process-consent-update extension returned non-200 status")
		return nil, fmt.Errorf("extension returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var extResponse models.PreProcessConsentUpdateResponse
	if err := json.Unmarshal(body, &extResponse); err != nil {
		c.logger.WithError(err).WithField("body", string(body)).Error("Failed to unmarshal pre-process-consent-update response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"responseId": extResponse.ResponseID,
		"status":     extResponse.Status,
	}).Debug("Pre-process-consent-update extension call completed")

	// Check if extension returned an error
	if extResponse.Status == "ERROR" {
		c.logger.WithFields(map[string]interface{}{
			"errorCode": extResponse.ErrorCode,
			"errorData": extResponse.ErrorData,
		}).Warn("Pre-process-consent-update extension returned ERROR status")
		// Return the response so caller can handle the error appropriately
		return &extResponse, nil
	}

	return &extResponse, nil
}

// EnrichConsentCreationResponse calls the enrich-consent-creation-response hook
// This enriches consent data after creation
func (c *ExtensionClient) EnrichConsentCreationResponse(ctx context.Context, consent *models.Consent) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.EnrichConsentCreationResponse == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequest(consent, nil)

	response, err := c.CallExtension(ctx, c.config.Endpoints.EnrichConsentCreationResponse, request)
	if err != nil {
		// Log error but don't fail the creation
		c.logger.WithError(err).Warn("enrich-consent-creation-response hook failed, continuing")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	return response, nil
}

// PreProcessConsentRetrieval calls the pre-process-consent-retrieval hook
// This is called before a consent is retrieved
func (c *ExtensionClient) PreProcessConsentRetrieval(ctx context.Context, consentID, clientID, orgID string) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentRetrieval == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, nil)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentRetrieval, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-retrieval hook failed: %w", err)
	}

	return response, nil
}

// EnrichConsentUpdateResponse calls the enrich-consent-update-response hook
// This is called after a consent is successfully updated
func (c *ExtensionClient) EnrichConsentUpdateResponse(ctx context.Context, consent *models.Consent) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.EnrichConsentUpdateResponse == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequest(consent, nil)

	response, err := c.CallExtension(ctx, c.config.Endpoints.EnrichConsentUpdateResponse, request)
	if err != nil {
		// Log error but don't fail the update
		c.logger.WithError(err).Warn("enrich-consent-update-response hook failed, continuing")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	return response, nil
}

// PreProcessConsentRevoke calls the pre-process-consent-revoke hook
// This is called before a consent is revoked
func (c *ExtensionClient) PreProcessConsentRevoke(ctx context.Context, consent *models.Consent) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentRevoke == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequest(consent, nil)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentRevoke, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-revoke hook failed: %w", err)
	}

	return response, nil
}

// PreProcessConsentFileUpload calls the pre-process-consent-file-upload hook
// This is called before a file is uploaded
func (c *ExtensionClient) PreProcessConsentFileUpload(ctx context.Context, consentID, clientID, orgID string, fileData map[string]interface{}) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentFileUpload == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, fileData)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentFileUpload, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-file-upload hook failed: %w", err)
	}

	return response, nil
}

// EnrichConsentFileResponse calls the enrich-consent-file-response hook
// This enriches the file response after upload
func (c *ExtensionClient) EnrichConsentFileResponse(ctx context.Context, consentID, clientID, orgID string, fileData map[string]interface{}) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.EnrichConsentFileResponse == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, fileData)

	response, err := c.CallExtension(ctx, c.config.Endpoints.EnrichConsentFileResponse, request)
	if err != nil {
		// Log error but don't fail
		c.logger.WithError(err).Warn("enrich-consent-file-response hook failed, continuing")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	return response, nil
}

// ValidateConsentFileRetrieval calls the validate-consent-file-retrieval hook
// This validates file retrieval requests
func (c *ExtensionClient) ValidateConsentFileRetrieval(ctx context.Context, consentID, clientID, orgID string) (*ValidationResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.ValidateConsentFileRetrieval == "" {
		return &ValidationResponse{Valid: true}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, nil)

	response, err := c.CallValidation(ctx, c.config.Endpoints.ValidateConsentFileRetrieval, request)
	if err != nil {
		return nil, fmt.Errorf("validate-consent-file-retrieval hook failed: %w", err)
	}

	return response, nil
}

// PreProcessConsentFileUpdate calls the pre-process-consent-file-update hook
// This is called before a file is updated
func (c *ExtensionClient) PreProcessConsentFileUpdate(ctx context.Context, consentID, clientID, orgID string, fileData map[string]interface{}) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentFileUpdate == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, fileData)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentFileUpdate, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-file-update hook failed: %w", err)
	}

	return response, nil
}

// EnrichConsentFileUpdateResponse calls the enrich-consent-file-update-response hook
// This enriches the file update response
func (c *ExtensionClient) EnrichConsentFileUpdateResponse(ctx context.Context, consentID, clientID, orgID string, fileData map[string]interface{}) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.EnrichConsentFileUpdateResponse == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequestFromMap(consentID, clientID, orgID, "", nil, fileData)

	response, err := c.CallExtension(ctx, c.config.Endpoints.EnrichConsentFileUpdateResponse, request)
	if err != nil {
		// Log error but don't fail
		c.logger.WithError(err).Warn("enrich-consent-file-update-response hook failed, continuing")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	return response, nil
}

// MapAcceleratorErrorResponse calls the map-accelerator-error-response hook
// This maps error responses from the accelerator
func (c *ExtensionClient) MapAcceleratorErrorResponse(ctx context.Context, errorCode, errorMessage string, additionalData map[string]interface{}) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.MapAcceleratorErrorResponse == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	if additionalData == nil {
		additionalData = make(map[string]interface{})
	}
	additionalData["errorCode"] = errorCode
	additionalData["errorMessage"] = errorMessage

	request := &ExtensionRequest{
		Data: additionalData,
	}

	response, err := c.CallExtension(ctx, c.config.Endpoints.MapAcceleratorErrorResponse, request)
	if err != nil {
		// Log error but don't fail
		c.logger.WithError(err).Warn("map-accelerator-error-response hook failed, continuing")
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	return response, nil
}
