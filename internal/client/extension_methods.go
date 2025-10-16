package client

import (
	"context"
	"fmt"

	"github.com/wso2/consent-management-api/internal/models"
)

// PreProcessConsentCreation calls the pre-process-consent-creation hook
// This is called before a consent is created to allow preprocessing
func (c *ExtensionClient) PreProcessConsentCreation(ctx context.Context, consent *models.Consent) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentCreation == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	request := BuildExtensionRequest(consent, nil)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentCreation, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-creation hook failed: %w", err)
	}

	return response, nil
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

// PreProcessConsentUpdate calls the pre-process-consent-update hook
// This is called before a consent is updated
func (c *ExtensionClient) PreProcessConsentUpdate(ctx context.Context, existingConsent, updatedConsent *models.Consent) (*ExtensionResponse, error) {
	if !c.IsExtensionEnabled() || c.config.Endpoints.PreProcessConsentUpdate == "" {
		return &ExtensionResponse{Success: true, Modified: false}, nil
	}

	// Include both existing and updated data
	additionalData := map[string]interface{}{
		"existingConsent": existingConsent,
		"updatedConsent":  updatedConsent,
	}

	request := BuildExtensionRequest(updatedConsent, additionalData)

	response, err := c.CallExtension(ctx, c.config.Endpoints.PreProcessConsentUpdate, request)
	if err != nil {
		return nil, fmt.Errorf("pre-process-consent-update hook failed: %w", err)
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
