package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const correlationIDKey contextKey = "correlationID"

// TestConsentIDGeneration tests ID generation uniqueness
func TestConsentIDGeneration(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := utils.GenerateConsentID()

		// Check format
		assert.Contains(t, id, "CONSENT-")

		// Check uniqueness
		assert.False(t, ids[id], "ID should be unique")
		ids[id] = true
	}

	assert.Equal(t, 100, len(ids), "Should have 100 unique IDs")
}

// TestConsentStatusValidation tests consent status validation
func TestConsentStatusValidation(t *testing.T) {
	validStatuses := []string{
		"CREATED",
		"AUTHORIZED",
		"ACTIVE",
		"REJECTED",
		"REVOKED",
		"EXPIRED",
	}

	for _, status := range validStatuses {
		err := utils.ValidateStatus(status)
		assert.NoError(t, err, "Status %s should be valid", status)
	}

	// Test empty status
	err := utils.ValidateStatus("")
	assert.Error(t, err, "Empty status should be invalid")

	// Test invalid status
	err = utils.ValidateStatus("INVALID")
	assert.Error(t, err, "Invalid status should be rejected")
}

// TestAuditRecordCreation tests audit trail creation
func TestAuditRecordCreation(t *testing.T) {
	consentID := "CONSENT-123"
	orgID := "ORG-001"
	previousStatus := "authorized"
	currentStatus := "revoked"
	reason := "User requested revocation"
	actionBy := "user-001"

	audit := &models.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateAuditID(),
		ConsentID:      consentID,
		CurrentStatus:  currentStatus,
		ActionTime:     utils.GetCurrentTimeMillis(),
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		OrgID:          orgID,
	}

	// Assertions
	assert.NotEmpty(t, audit.StatusAuditID)
	assert.Contains(t, audit.StatusAuditID, "AUDIT-")
	assert.Equal(t, consentID, audit.ConsentID)
	assert.Equal(t, currentStatus, audit.CurrentStatus)
	assert.Equal(t, previousStatus, *audit.PreviousStatus)
	assert.Equal(t, reason, *audit.Reason)
	assert.Equal(t, actionBy, *audit.ActionBy)
	assert.NotZero(t, audit.ActionTime)
}

// TestBuildConsentObject tests consent object creation from request
func TestBuildConsentObject(t *testing.T) {
	receiptData := map[string]interface{}{
		"consentId": "test-123",
		"data":      "sample",
	}

	request := &models.ConsentCreateRequest{
		Receipt:            receiptData,
		ConsentType:        "accounts",
		CurrentStatus:      "authorized",
		ConsentFrequency:   nil,
		ValidityTime:       nil,
		RecurringIndicator: nil,
	}

	orgID := "ORG-001"
	clientID := "client-001"
	consentID := utils.GenerateConsentID()
	currentTime := utils.GetCurrentTimeMillis()

	// Convert receipt to JSON for consent model
	receiptJSON, err := json.Marshal(request.Receipt)
	assert.NoError(t, err)

	// Build consent object (clientID comes from context, not request)
	consent := &models.Consent{
		ConsentID:          consentID,
		Receipt:            receiptJSON,
		CreatedTime:        currentTime,
		UpdatedTime:        currentTime,
		ClientID:           clientID,
		ConsentType:        request.ConsentType,
		CurrentStatus:      request.CurrentStatus,
		ConsentFrequency:   request.ConsentFrequency,
		ValidityTime:       request.ValidityTime,
		RecurringIndicator: request.RecurringIndicator,
		OrgID:              orgID,
	}

	// Assertions
	assert.NotEmpty(t, consent.ConsentID)
	assert.Contains(t, consent.ConsentID, "CONSENT-")
	assert.Equal(t, clientID, consent.ClientID)
	assert.Equal(t, request.ConsentType, consent.ConsentType)
	assert.Equal(t, orgID, consent.OrgID)
	assert.Equal(t, "authorized", consent.CurrentStatus)
	assert.NotZero(t, consent.CreatedTime)
	assert.NotZero(t, consent.UpdatedTime)
}

// TestValidationFunctions tests various validation utilities
func TestValidationFunctions(t *testing.T) {
	t.Run("ValidateClientID", func(t *testing.T) {
		// Valid client ID
		err := utils.ValidateClientID("client-001")
		assert.NoError(t, err)

		// Empty client ID
		err = utils.ValidateClientID("")
		assert.Error(t, err)
	})

	t.Run("ValidateOrgID", func(t *testing.T) {
		// Valid org ID
		err := utils.ValidateOrgID("ORG-001")
		assert.NoError(t, err)

		// Empty org ID
		err = utils.ValidateOrgID("")
		assert.Error(t, err)
	})

	t.Run("ValidateConsentType", func(t *testing.T) {
		// Valid consent type
		err := utils.ValidateConsentType("accounts")
		assert.NoError(t, err)

		// Empty consent type
		err = utils.ValidateConsentType("")
		assert.Error(t, err)
	})
}

// TestContextHandling tests context propagation
func TestContextHandling(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, correlationIDKey, "test-correlation-123")

	// Verify context value
	correlationID, ok := ctx.Value(correlationIDKey).(string)
	assert.True(t, ok, "Context should contain correlationID")
	assert.Equal(t, "test-correlation-123", correlationID)
}

// TestTimeUtilities tests time-related helper functions
func TestTimeUtilities(t *testing.T) {
	t.Run("GetCurrentTimeMillis", func(t *testing.T) {
		time1 := utils.GetCurrentTimeMillis()
		assert.Greater(t, time1, int64(0))

		time2 := utils.GetCurrentTimeMillis()
		assert.GreaterOrEqual(t, time2, time1)
	})

	t.Run("TimeConversion", func(t *testing.T) {
		millis := utils.GetCurrentTimeMillis()
		timeObj := utils.MillisToTime(millis)
		assert.NotNil(t, timeObj)

		convertedBack := utils.TimeToMillis(timeObj)
		assert.Equal(t, millis, convertedBack)
	})
}
