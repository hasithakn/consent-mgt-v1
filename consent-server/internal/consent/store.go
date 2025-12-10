package consent

import (
	"context"

	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/system/database"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
)

// DBQuery objects for consent operations
var (
	QueryCreateConsent = dbmodel.DBQuery{
		ID:    "CREATE_CONSENT",
		Query: "INSERT INTO CONSENT (CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetConsentByID = dbmodel.DBQuery{
		ID:    "GET_CONSENT_BY_ID",
		Query: "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryListConsents = dbmodel.DBQuery{
		ID:    "LIST_CONSENTS",
		Query: "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE ORG_ID = ? ORDER BY CREATED_TIME DESC LIMIT ? OFFSET ?",
	}

	QueryCountConsents = dbmodel.DBQuery{
		ID:    "COUNT_CONSENTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT WHERE ORG_ID = ?",
	}

	QueryUpdateConsent = dbmodel.DBQuery{
		ID:    "UPDATE_CONSENT",
		Query: "UPDATE CONSENT SET UPDATED_TIME = ?, CONSENT_TYPE = ?, CONSENT_FREQUENCY = ?, VALIDITY_TIME = ?, RECURRING_INDICATOR = ?, DATA_ACCESS_VALIDITY_DURATION = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateConsentStatus = dbmodel.DBQuery{
		ID:    "UPDATE_CONSENT_STATUS",
		Query: "UPDATE CONSENT SET CURRENT_STATUS = ?, UPDATED_TIME = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteConsent = dbmodel.DBQuery{
		ID:    "DELETE_CONSENT",
		Query: "DELETE FROM CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryGetConsentsByClientID = dbmodel.DBQuery{
		ID:    "GET_CONSENTS_BY_CLIENT_ID",
		Query: "SELECT CONSENT_ID, CREATED_TIME, UPDATED_TIME, CLIENT_ID, CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME, RECURRING_INDICATOR, DATA_ACCESS_VALIDITY_DURATION, ORG_ID FROM CONSENT WHERE CLIENT_ID = ? AND ORG_ID = ?",
	}

	// Attribute queries
	QueryCreateAttribute = dbmodel.DBQuery{
		ID:    "CREATE_CONSENT_ATTRIBUTE",
		Query: "INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID) VALUES (?, ?, ?, ?)",
	}

	QueryGetAttributesByConsentID = dbmodel.DBQuery{
		ID:    "GET_ATTRIBUTES_BY_CONSENT_ID",
		Query: "SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAttributesByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_ATTRIBUTES_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	// Status audit queries
	QueryCreateStatusAudit = dbmodel.DBQuery{
		ID:    "CREATE_STATUS_AUDIT",
		Query: "INSERT INTO CONSENT_STATUS_AUDIT (STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME, REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetStatusAuditByConsentID = dbmodel.DBQuery{
		ID:    "GET_STATUS_AUDIT_BY_CONSENT_ID",
		Query: "SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME, REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID FROM CONSENT_STATUS_AUDIT WHERE CONSENT_ID = ? AND ORG_ID = ? ORDER BY ACTION_TIME DESC",
	}
)

// consentStore defines the interface for consent data operations
type consentStore interface {
	// Consent operations
	Create(ctx context.Context, consent *model.Consent) error
	GetByID(ctx context.Context, consentID, orgID string) (*model.Consent, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]model.Consent, int, error)
	Update(ctx context.Context, consent *model.Consent) error
	UpdateStatus(ctx context.Context, consentID, orgID, status string, updatedTime int64) error
	Delete(ctx context.Context, consentID, orgID string) error
	GetByClientID(ctx context.Context, clientID, orgID string) ([]model.Consent, error)

	// Attribute operations
	CreateAttributes(ctx context.Context, attributes []model.ConsentAttribute) error
	GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error)
	DeleteAttributesByConsentID(ctx context.Context, consentID, orgID string) error

	// Status audit operations
	CreateStatusAudit(ctx context.Context, audit *model.ConsentStatusAudit) error
	GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentStatusAudit, error)

	// Transactional operations
	CreateWithTx(ctx context.Context, tx *database.Tx, consent *model.Consent) error
	CreateAttributesWithTx(ctx context.Context, tx *database.Tx, attributes []model.ConsentAttribute) error
	CreateStatusAuditWithTx(ctx context.Context, tx *database.Tx, audit *model.ConsentStatusAudit) error
}

// store implements the consentStore interface
type store struct {
	dbClient provider.DBClientInterface
}

// newConsentStore creates a new consent store
func newConsentStore(dbClient provider.DBClientInterface) consentStore {
	return &store{
		dbClient: dbClient,
	}
}

// Create creates a new consent
func (s *store) Create(ctx context.Context, consent *model.Consent) error {
	_, err := s.dbClient.Execute(QueryCreateConsent,
		consent.ConsentID, consent.CreatedTime, consent.UpdatedTime, consent.ClientID,
		consent.ConsentType, consent.CurrentStatus, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.OrgID)
	return err
}

// GetByID retrieves a consent by ID
func (s *store) GetByID(ctx context.Context, consentID, orgID string) (*model.Consent, error) {
	rows, err := s.dbClient.Query(QueryGetConsentByID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsent(rows[0]), nil
}

// List retrieves paginated consents
func (s *store) List(ctx context.Context, orgID string, limit, offset int) ([]model.Consent, int, error) {
	countRows, err := s.dbClient.Query(QueryCountConsents, orgID)
	if err != nil {
		return nil, 0, err
	}

	totalCount := 0
	if len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			totalCount = int(count)
		}
	}

	rows, err := s.dbClient.Query(QueryListConsents, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	consents := make([]model.Consent, 0, len(rows))
	for _, row := range rows {
		consent := mapToConsent(row)
		if consent != nil {
			consents = append(consents, *consent)
		}
	}

	return consents, totalCount, nil
}

// Update updates a consent
func (s *store) Update(ctx context.Context, consent *model.Consent) error {
	_, err := s.dbClient.Execute(QueryUpdateConsent,
		consent.UpdatedTime, consent.ConsentType, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.ConsentID, consent.OrgID)
	return err
}

// UpdateStatus updates consent status
func (s *store) UpdateStatus(ctx context.Context, consentID, orgID, status string, updatedTime int64) error {
	_, err := s.dbClient.Execute(QueryUpdateConsentStatus, status, updatedTime, consentID, orgID)
	return err
}

// Delete deletes a consent
func (s *store) Delete(ctx context.Context, consentID, orgID string) error {
	_, err := s.dbClient.Execute(QueryDeleteConsent, consentID, orgID)
	return err
}

// GetByClientID retrieves consents by client ID
func (s *store) GetByClientID(ctx context.Context, clientID, orgID string) ([]model.Consent, error) {
	rows, err := s.dbClient.Query(QueryGetConsentsByClientID, clientID, orgID)
	if err != nil {
		return nil, err
	}

	consents := make([]model.Consent, 0, len(rows))
	for _, row := range rows {
		consent := mapToConsent(row)
		if consent != nil {
			consents = append(consents, *consent)
		}
	}

	return consents, nil
}

// CreateAttributes creates multiple consent attributes
func (s *store) CreateAttributes(ctx context.Context, attributes []model.ConsentAttribute) error {
	for _, attr := range attributes {
		_, err := s.dbClient.Execute(QueryCreateAttribute,
			attr.ConsentID, attr.AttKey, attr.AttValue, attr.OrgID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAttributesByConsentID retrieves attributes for a consent
func (s *store) GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error) {
	rows, err := s.dbClient.Query(QueryGetAttributesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	attributes := make([]model.ConsentAttribute, 0, len(rows))
	for _, row := range rows {
		attr := mapToConsentAttribute(row)
		if attr != nil {
			attributes = append(attributes, *attr)
		}
	}

	return attributes, nil
}

// DeleteAttributesByConsentID deletes all attributes for a consent
func (s *store) DeleteAttributesByConsentID(ctx context.Context, consentID, orgID string) error {
	_, err := s.dbClient.Execute(QueryDeleteAttributesByConsentID, consentID, orgID)
	return err
}

// CreateStatusAudit creates a status audit entry
func (s *store) CreateStatusAudit(ctx context.Context, audit *model.ConsentStatusAudit) error {
	_, err := s.dbClient.Execute(QueryCreateStatusAudit,
		audit.StatusAuditID, audit.ConsentID, audit.CurrentStatus, audit.ActionTime,
		audit.Reason, audit.ActionBy, audit.PreviousStatus, audit.OrgID)
	return err
}

// GetStatusAuditByConsentID retrieves status audit history for a consent
func (s *store) GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentStatusAudit, error) {
	rows, err := s.dbClient.Query(QueryGetStatusAuditByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	audits := make([]model.ConsentStatusAudit, 0, len(rows))
	for _, row := range rows {
		audit := mapToStatusAudit(row)
		if audit != nil {
			audits = append(audits, *audit)
		}
	}

	return audits, nil
}

// Mapper functions

func mapToConsent(row map[string]interface{}) *model.Consent {
	if row == nil {
		return nil
	}

	consent := &model.Consent{}

	if id, ok := row["CONSENT_ID"].(string); ok {
		consent.ConsentID = id
	}
	if created, ok := row["CREATED_TIME"].(int64); ok {
		consent.CreatedTime = created
	}
	if updated, ok := row["UPDATED_TIME"].(int64); ok {
		consent.UpdatedTime = updated
	}
	if clientID, ok := row["CLIENT_ID"].(string); ok {
		consent.ClientID = clientID
	}
	if cType, ok := row["CONSENT_TYPE"].(string); ok {
		consent.ConsentType = cType
	}
	if status, ok := row["CURRENT_STATUS"].(string); ok {
		consent.CurrentStatus = status
	}
	if freq, ok := row["CONSENT_FREQUENCY"].(int64); ok {
		freqInt := int(freq)
		consent.ConsentFrequency = &freqInt
	}
	if valid, ok := row["VALIDITY_TIME"].(int64); ok {
		consent.ValidityTime = &valid
	}
	if recurring, ok := row["RECURRING_INDICATOR"].(bool); ok {
		consent.RecurringIndicator = &recurring
	}
	if duration, ok := row["DATA_ACCESS_VALIDITY_DURATION"].(int64); ok {
		consent.DataAccessValidityDuration = &duration
	}
	if orgID, ok := row["ORG_ID"].(string); ok {
		consent.OrgID = orgID
	}

	return consent
}

func mapToConsentAttribute(row map[string]interface{}) *model.ConsentAttribute {
	if row == nil {
		return nil
	}

	attr := &model.ConsentAttribute{}

	if consentID, ok := row["CONSENT_ID"].(string); ok {
		attr.ConsentID = consentID
	}
	if key, ok := row["ATT_KEY"].(string); ok {
		attr.AttKey = key
	}
	if value, ok := row["ATT_VALUE"].(string); ok {
		attr.AttValue = value
	}
	if orgID, ok := row["ORG_ID"].(string); ok {
		attr.OrgID = orgID
	}

	return attr
}

func mapToStatusAudit(row map[string]interface{}) *model.ConsentStatusAudit {
	if row == nil {
		return nil
	}

	audit := &model.ConsentStatusAudit{}

	if id, ok := row["STATUS_AUDIT_ID"].(string); ok {
		audit.StatusAuditID = id
	}
	if consentID, ok := row["CONSENT_ID"].(string); ok {
		audit.ConsentID = consentID
	}
	if status, ok := row["CURRENT_STATUS"].(string); ok {
		audit.CurrentStatus = status
	}
	if actionTime, ok := row["ACTION_TIME"].(int64); ok {
		audit.ActionTime = actionTime
	}
	if reason, ok := row["REASON"].(string); ok {
		audit.Reason = &reason
	}
	if actionBy, ok := row["ACTION_BY"].(string); ok {
		audit.ActionBy = &actionBy
	}
	if prevStatus, ok := row["PREVIOUS_STATUS"].(string); ok {
		audit.PreviousStatus = &prevStatus
	}
	if orgID, ok := row["ORG_ID"].(string); ok {
		audit.OrgID = orgID
	}

	return audit
}

// CreateWithTx creates a consent within a transaction
func (s *store) CreateWithTx(ctx context.Context, tx *database.Tx, consent *model.Consent) error {
	_, err := tx.ExecContext(ctx, QueryCreateConsent.Query,
		consent.ConsentID, consent.CreatedTime, consent.UpdatedTime, consent.ClientID,
		consent.ConsentType, consent.CurrentStatus, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.OrgID)
	return err
}

// CreateAttributesWithTx creates consent attributes within a transaction
func (s *store) CreateAttributesWithTx(ctx context.Context, tx *database.Tx, attributes []model.ConsentAttribute) error {
	for _, attr := range attributes {
		_, err := tx.ExecContext(ctx, QueryCreateAttribute.Query,
			attr.ConsentID, attr.AttKey, attr.AttValue, attr.OrgID)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateStatusAuditWithTx creates a status audit record within a transaction
func (s *store) CreateStatusAuditWithTx(ctx context.Context, tx *database.Tx, audit *model.ConsentStatusAudit) error {
	_, err := tx.ExecContext(ctx, QueryCreateStatusAudit.Query,
		audit.StatusAuditID, audit.ConsentID, audit.CurrentStatus, audit.ActionTime,
		audit.Reason, audit.ActionBy, audit.PreviousStatus, audit.OrgID)
	return err
}
