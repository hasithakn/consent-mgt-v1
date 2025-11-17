package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/internal/utils"
)

// ConsentPurposeHandler handles consent purpose-related HTTP requests
type ConsentPurposeHandler struct {
	purposeService *service.ConsentPurposeService
}

// NewConsentPurposeHandler creates a new consent purpose handler instance
func NewConsentPurposeHandler(purposeService *service.ConsentPurposeService) *ConsentPurposeHandler {
	return &ConsentPurposeHandler{
		purposeService: purposeService,
	}
}

// CreateConsentPurposes handles POST /consent-purposes
// Creates one or more consent purposes in batch (transactional - all or nothing)
func (h *ConsentPurposeHandler) CreateConsentPurposes(c *gin.Context) {
	// Parse request body - array of purpose requests
	var requests []models.ConsentPurposeCreateRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Validate that we have at least one purpose to create
	if len(requests) == 0 {
		utils.SendBadRequestError(c, "Empty request", "At least one purpose must be provided")
		return
	}

	// Get orgID from context (set by middleware)
	orgID := utils.GetOrgIDFromContext(c)

	// Convert to service requests
	serviceRequests := make([]*service.ConsentPurposeCreateRequest, 0, len(requests))
	for _, request := range requests {
		// Convert to service request
		var desc *string
		if request.Description != "" {
			desc = &request.Description
		}

		serviceReq := &service.ConsentPurposeCreateRequest{
			Name:        request.Name,
			Description: desc,
			Type:        request.Type,
			Attributes:  request.Attributes,
		}
		serviceRequests = append(serviceRequests, serviceReq)
	}

	// Create all purposes in a transaction (all or nothing)
	createdPurposes, err := h.purposeService.CreatePurposesInBatch(c.Request.Context(), orgID, serviceRequests)
	if err != nil {
		// Check if it's an attribute validation error
		if strings.Contains(err.Error(), "attribute validation failed") {
			// Return structured validation error
			c.JSON(400, gin.H{
				"error":   "Validation failed",
				"message": err.Error(),
				"type":    "attribute_validation_error",
			})
			return
		}
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid purpose type") ||
			strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "duplicate") ||
			strings.Contains(err.Error(), "invalid request") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		utils.SendInternalServerError(c, "Failed to create consent purposes", err.Error())
		return
	}

	// Return created purposes with 201 Created status
	c.JSON(201, gin.H{
		"data":    createdPurposes,
		"message": "Consent purposes created successfully",
	})
}

// GetConsentPurpose handles GET /consent-purposes/:purposeId
func (h *ConsentPurposeHandler) GetConsentPurpose(c *gin.Context) {
	purposeID := c.Param("purposeId")
	if purposeID == "" {
		utils.SendBadRequestError(c, "Purpose ID is required", "")
		return
	}

	orgID := utils.GetOrgIDFromContext(c)

	// Get the purpose
	response, err := h.purposeService.GetPurpose(c.Request.Context(), purposeID, orgID)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Consent purpose not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to retrieve consent purpose", err.Error())
		return
	}

	utils.SendOKResponse(c, response)
}

// ListConsentPurposes handles GET /consent-purposes
func (h *ConsentPurposeHandler) ListConsentPurposes(c *gin.Context) {
	orgID := utils.GetOrgIDFromContext(c)

	// Get pagination parameters from query string
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Validate pagination parameters
	if limit < 1 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// List purposes
	response, err := h.purposeService.ListPurposes(c.Request.Context(), orgID, limit, offset)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to list consent purposes", err.Error())
		return
	}

	utils.SendOKResponse(c, response)
}

// UpdateConsentPurpose handles PUT /consent-purposes/:purposeId
func (h *ConsentPurposeHandler) UpdateConsentPurpose(c *gin.Context) {
	purposeID := c.Param("purposeId")
	if purposeID == "" {
		utils.SendBadRequestError(c, "Purpose ID is required", "")
		return
	}

	orgID := utils.GetOrgIDFromContext(c)

	// Parse request body
	var request models.ConsentPurposeUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Convert to service request (all fields are required, no partial updates)
	serviceReq := &service.ConsentPurposeUpdateRequest{
		Name:        request.Name,
		Description: request.Description,
		Type:        request.Type,
		Attributes:  request.Attributes,
	}

	// Update the purpose
	response, err := h.purposeService.UpdatePurpose(c.Request.Context(), purposeID, orgID, serviceReq)
	if err != nil {
		// Check if it's an attribute validation error
		if strings.Contains(err.Error(), "attribute validation failed") {
			// Return structured validation error
			c.JSON(400, gin.H{
				"error":   "Validation failed",
				"message": err.Error(),
				"type":    "attribute_validation_error",
			})
			return
		}
		// Check if it's a validation error
		if strings.Contains(err.Error(), "is required") ||
			strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid purpose type") ||
			strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "currently used by") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Consent purpose not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to update consent purpose", err.Error())
		return
	}

	utils.SendOKResponse(c, response)
}

// DeleteConsentPurpose handles DELETE /consent-purposes/:purposeId
func (h *ConsentPurposeHandler) DeleteConsentPurpose(c *gin.Context) {
	purposeID := c.Param("purposeId")
	if purposeID == "" {
		utils.SendBadRequestError(c, "Purpose ID is required", "")
		return
	}

	orgID := utils.GetOrgIDFromContext(c)

	// Delete the purpose
	err := h.purposeService.DeletePurpose(c.Request.Context(), purposeID, orgID)
	if err != nil {
		// Check if it's a binding constraint error
		if strings.Contains(err.Error(), "currently used by") {
			utils.SendBadRequestError(c, "Cannot delete consent purpose", err.Error())
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Consent purpose not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to delete consent purpose", err.Error())
		return
	}

	// Return 204 No Content on successful deletion
	c.Status(204)
}

// ValidateConsentPurposes handles GET /consent-purposes/validate
// Validates a list of purpose names and returns only the valid ones that exist
func (h *ConsentPurposeHandler) ValidateConsentPurposes(c *gin.Context) {
	orgID := utils.GetOrgIDFromContext(c)

	// Parse request body - array of purpose names
	var purposeNames []string
	if err := c.ShouldBindJSON(&purposeNames); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Validate purpose names and get valid ones
	validNames, err := h.purposeService.ValidatePurposeNames(c.Request.Context(), orgID, purposeNames)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "is required") ||
			strings.Contains(err.Error(), "must be provided") ||
			strings.Contains(err.Error(), "no valid purposes found") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		utils.SendInternalServerError(c, "Failed to validate purpose names", err.Error())
		return
	}

	// Return valid purpose names with 200 OK
	c.JSON(200, validNames)
}
