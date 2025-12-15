package consent

import (
	"context"
	"fmt"
	"strings"

	"github.com/wso2/consent-management-api/internal/consent/model"
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

	QueryFindConsentIDsByAttributeKey = dbmodel.DBQuery{
		ID:    "FIND_CONSENT_IDS_BY_ATTRIBUTE_KEY",
		Query: "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = ? AND ORG_ID = ? ORDER BY CONSENT_ID",
	}

	QueryFindConsentIDsByAttribute = dbmodel.DBQuery{
		ID:    "FIND_CONSENT_IDS_BY_ATTRIBUTE",
		Query: "SELECT DISTINCT CONSENT_ID FROM CONSENT_ATTRIBUTE WHERE ATT_KEY = ? AND ATT_VALUE = ? AND ORG_ID = ? ORDER BY CONSENT_ID",
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

	QueryGetAttributesByConsentIDs = dbmodel.DBQuery{
		ID:    "GET_ATTRIBUTES_BY_CONSENT_IDS",
		Query: "", // Built dynamically
	}

	QuerySearchConsents = dbmodel.DBQuery{
		ID:    "SEARCH_CONSENTS",
		Query: "", // Built dynamically
	}
)

// consentStore defines the interface for consent data operations
// ConsentStore defines the interface for consent data access operations
type ConsentStore interface {
	// Read operations - use dbClient directly
	GetByID(ctx context.Context, consentID, orgID string) (*model.Consent, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]model.Consent, int, error)
	Search(ctx context.Context, filters model.ConsentSearchFilters) ([]model.Consent, int, error)
	GetByClientID(ctx context.Context, clientID, orgID string) ([]model.Consent, error)
	GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentAttribute, error)
	GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error)
	GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentStatusAudit, error)
	FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error)
	FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error)

	// Write operations - transactional with tx parameter
	Create(tx dbmodel.TxInterface, consent *model.Consent) error
	Update(tx dbmodel.TxInterface, consent *model.Consent) error
	UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
	Delete(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error
	DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error
}

// store implements the ConsentStore interface
type store struct {
	dbClient provider.DBClientInterface
}

// NewConsentStore creates a new consent store
func NewConsentStore(dbClient provider.DBClientInterface) ConsentStore {
	return &store{
		dbClient: dbClient,
	}
}

// Create creates a new consent within a transaction
func (s *store) Create(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryCreateConsent.Query,
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

// Search retrieves consents based on filters with pagination
func (s *store) Search(ctx context.Context, filters model.ConsentSearchFilters) ([]model.Consent, int, error) {
	// Build WHERE clause dynamically
	whereConditions := []string{"CONSENT.ORG_ID = ?"}
	args := []interface{}{filters.OrgID}
	countArgs := []interface{}{filters.OrgID}

	// Add consentTypes filter (IN clause)
	if len(filters.ConsentTypes) > 0 {
		placeholders := make([]string, len(filters.ConsentTypes))
		for i, ct := range filters.ConsentTypes {
			placeholders[i] = "?"
			args = append(args, ct)
			countArgs = append(countArgs, ct)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CONSENT_TYPE IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add consentStatuses filter (IN clause) - convert to uppercase
	if len(filters.ConsentStatuses) > 0 {
		placeholders := make([]string, len(filters.ConsentStatuses))
		for i, status := range filters.ConsentStatuses {
			placeholders[i] = "?"
			// Convert to uppercase to match DB values (ACTIVE, REJECTED, etc.)
			args = append(args, strings.ToUpper(status))
			countArgs = append(countArgs, strings.ToUpper(status))
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CURRENT_STATUS IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add clientIds filter (IN clause)
	if len(filters.ClientIDs) > 0 {
		placeholders := make([]string, len(filters.ClientIDs))
		for i, clientID := range filters.ClientIDs {
			placeholders[i] = "?"
			args = append(args, clientID)
			countArgs = append(countArgs, clientID)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("CONSENT.CLIENT_ID IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add userIds filter (via JOIN with CONSENT_AUTH_RESOURCE)
	joinClause := ""
	if len(filters.UserIDs) > 0 {
		placeholders := make([]string, len(filters.UserIDs))
		for i, userID := range filters.UserIDs {
			placeholders[i] = "?"
			args = append(args, userID)
			countArgs = append(countArgs, userID)
		}
		joinClause = " INNER JOIN CONSENT_AUTH_RESOURCE car ON CONSENT.CONSENT_ID = car.CONSENT_ID AND CONSENT.ORG_ID = car.ORG_ID"
		whereConditions = append(whereConditions, fmt.Sprintf("car.USER_ID IN (%s)", strings.Join(placeholders, ",")))
	}

	// Add time range filters (timestamps in milliseconds)
	if filters.FromTime != nil {
		whereConditions = append(whereConditions, "CONSENT.CREATED_TIME >= ?")
		args = append(args, *filters.FromTime)
		countArgs = append(countArgs, *filters.FromTime)
	}

	if filters.ToTime != nil {
		whereConditions = append(whereConditions, "CONSENT.CREATED_TIME <= ?")
		args = append(args, *filters.ToTime)
		countArgs = append(countArgs, *filters.ToTime)
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Build COUNT query
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT CONSENT.CONSENT_ID) as count FROM CONSENT%s WHERE %s",
		joinClause, whereClause)

	// Execute count query
	countRows, err := s.dbClient.Query(dbmodel.DBQuery{ID: "COUNT_SEARCH_RESULTS", Query: countQuery}, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	totalCount := 0
	if len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			totalCount = int(count)
		}
	}

	// Build SELECT query with DISTINCT to handle JOIN duplicates
	selectQuery := fmt.Sprintf(
		"SELECT DISTINCT CONSENT.CONSENT_ID, CONSENT.CREATED_TIME, CONSENT.UPDATED_TIME, CONSENT.CLIENT_ID, CONSENT.CONSENT_TYPE, CONSENT.CURRENT_STATUS, CONSENT.CONSENT_FREQUENCY, CONSENT.VALIDITY_TIME, CONSENT.RECURRING_INDICATOR, CONSENT.DATA_ACCESS_VALIDITY_DURATION, CONSENT.ORG_ID FROM CONSENT%s WHERE %s ORDER BY CONSENT.CREATED_TIME DESC LIMIT ? OFFSET ?",
		joinClause,
		whereClause,
	)

	// Add pagination parameters
	args = append(args, filters.Limit, filters.Offset)

	// Execute search query
	rows, err := s.dbClient.Query(dbmodel.DBQuery{ID: "SEARCH_CONSENTS", Query: selectQuery}, args...)
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

// Update updates a consent within a transaction
func (s *store) Update(tx dbmodel.TxInterface, consent *model.Consent) error {
	_, err := tx.Exec(QueryUpdateConsent.Query,
		consent.UpdatedTime, consent.ConsentType, consent.ConsentFrequency,
		consent.ValidityTime, consent.RecurringIndicator, consent.DataAccessValidityDuration,
		consent.ConsentID, consent.OrgID)
	return err
}

// UpdateStatus updates consent status within a transaction
func (s *store) UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error {
	result, err := tx.Exec(QueryUpdateConsentStatus.Query, status, updatedTime, consentID, orgID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no consent found with CONSENT_ID=%s and ORG_ID=%s", consentID, orgID)
	}

	return nil
}

// Delete deletes a consent within a transaction
func (s *store) Delete(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteConsent.Query, consentID, orgID)
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

// CreateAttributes creates multiple consent attributes within a transaction
func (s *store) CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentAttribute) error {
	for _, attr := range attributes {
		_, err := tx.Exec(QueryCreateAttribute.Query,
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

// GetAttributesByConsentIDs retrieves attributes for multiple consents, grouped by consent ID
func (s *store) GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error) {
	if len(consentIDs) == 0 {
		return make(map[string]map[string]string), nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, 0, len(consentIDs)+1)
	for i, id := range consentIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, id)
	}
	args = append(args, orgID)

	// Build dynamic query
	query := dbmodel.DBQuery{
		ID:    QueryGetAttributesByConsentIDs.ID,
		Query: fmt.Sprintf("SELECT CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID IN (%s) AND ORG_ID = ?", placeholders),
	}

	rows, err := s.dbClient.Query(query, args...)
	if err != nil {
		return nil, err
	}

	// Group attributes by consent ID
	result := make(map[string]map[string]string)
	for _, row := range rows {
		attr := mapToConsentAttribute(row)
		if attr != nil {
			if result[attr.ConsentID] == nil {
				result[attr.ConsentID] = make(map[string]string)
			}
			result[attr.ConsentID][attr.AttKey] = attr.AttValue
		}
	}

	return result, nil
}

// DeleteAttributesByConsentID deletes all attributes for a consent within a transaction
func (s *store) DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAttributesByConsentID.Query, consentID, orgID)
	return err
}

// FindConsentIDsByAttributeKey finds all consent IDs that have a specific attribute key
func (s *store) FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error) {
	rows, err := s.dbClient.Query(QueryFindConsentIDsByAttributeKey, key, orgID)
	if err != nil {
		return nil, err
	}

	consentIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		// Try lowercase first (normalized), then uppercase (raw)
		if consentID, ok := row["consent_id"].(string); ok {
			consentIDs = append(consentIDs, consentID)
		} else if consentID, ok := row["consent_id"].([]byte); ok {
			consentIDs = append(consentIDs, string(consentID))
		} else if consentID, ok := row["CONSENT_ID"].(string); ok {
			consentIDs = append(consentIDs, consentID)
		} else if consentID, ok := row["CONSENT_ID"].([]byte); ok {
			consentIDs = append(consentIDs, string(consentID))
		}
	}

	return consentIDs, nil
}

// FindConsentIDsByAttribute finds all consent IDs that have a specific attribute key-value pair
func (s *store) FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error) {
	rows, err := s.dbClient.Query(QueryFindConsentIDsByAttribute, key, value, orgID)
	if err != nil {
		return nil, err
	}

	consentIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		// Try lowercase first (normalized), then uppercase (raw)
		if consentID, ok := row["consent_id"].(string); ok {
			consentIDs = append(consentIDs, consentID)
		} else if consentID, ok := row["consent_id"].([]byte); ok {
			consentIDs = append(consentIDs, string(consentID))
		} else if consentID, ok := row["CONSENT_ID"].(string); ok {
			consentIDs = append(consentIDs, consentID)
		} else if consentID, ok := row["CONSENT_ID"].([]byte); ok {
			consentIDs = append(consentIDs, string(consentID))
		}
	}

	return consentIDs, nil
}

// CreateStatusAudit creates a status audit entry within a transaction
func (s *store) CreateStatusAudit(tx dbmodel.TxInterface, audit *model.ConsentStatusAudit) error {
	_, err := tx.Exec(QueryCreateStatusAudit.Query,
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

// mapToConsent converts a database row map to Consent
// Note: DBClient normalizes column names to lowercase
func mapToConsent(row map[string]interface{}) *model.Consent {
	if row == nil {
		return nil
	}

	consent := &model.Consent{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["consent_id"].(string); ok {
		consent.ConsentID = id
	} else if id, ok := row["consent_id"].([]byte); ok {
		consent.ConsentID = string(id)
	}

	if created, ok := row["created_time"].(int64); ok {
		consent.CreatedTime = created
	}

	if updated, ok := row["updated_time"].(int64); ok {
		consent.UpdatedTime = updated
	}

	if clientID, ok := row["client_id"].(string); ok {
		consent.ClientID = clientID
	} else if clientID, ok := row["client_id"].([]byte); ok {
		consent.ClientID = string(clientID)
	}

	if cType, ok := row["consent_type"].(string); ok {
		consent.ConsentType = cType
	} else if cType, ok := row["consent_type"].([]byte); ok {
		consent.ConsentType = string(cType)
	}

	if status, ok := row["current_status"].(string); ok {
		consent.CurrentStatus = status
	} else if status, ok := row["current_status"].([]byte); ok {
		consent.CurrentStatus = string(status)
	}

	if freq, ok := row["consent_frequency"].(int64); ok {
		freqInt := int(freq)
		consent.ConsentFrequency = &freqInt
	}

	if valid, ok := row["validity_time"].(int64); ok {
		consent.ValidityTime = &valid
	}

	if recurring, ok := row["recurring_indicator"].(bool); ok {
		consent.RecurringIndicator = &recurring
	} else if recurring, ok := row["recurring_indicator"].(int64); ok {
		recurringBool := recurring != 0
		consent.RecurringIndicator = &recurringBool
	}

	if duration, ok := row["data_access_validity_duration"].(int64); ok {
		consent.DataAccessValidityDuration = &duration
	}

	if orgID, ok := row["org_id"].(string); ok {
		consent.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		consent.OrgID = string(orgID)
	}

	return consent
}

// mapToConsentAttribute converts a database row map to ConsentAttribute
// Note: DBClient normalizes column names to lowercase
func mapToConsentAttribute(row map[string]interface{}) *model.ConsentAttribute {
	if row == nil {
		return nil
	}

	attr := &model.ConsentAttribute{}

	// Handle string columns (may be string or []byte from MySQL)
	if consentID, ok := row["consent_id"].(string); ok {
		attr.ConsentID = consentID
	} else if consentID, ok := row["consent_id"].([]byte); ok {
		attr.ConsentID = string(consentID)
	}

	if key, ok := row["att_key"].(string); ok {
		attr.AttKey = key
	} else if key, ok := row["att_key"].([]byte); ok {
		attr.AttKey = string(key)
	}

	if value, ok := row["att_value"].(string); ok {
		attr.AttValue = value
	} else if value, ok := row["att_value"].([]byte); ok {
		attr.AttValue = string(value)
	}

	if orgID, ok := row["org_id"].(string); ok {
		attr.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		attr.OrgID = string(orgID)
	}

	return attr
}

// mapToStatusAudit converts a database row map to ConsentStatusAudit
// Note: DBClient normalizes column names to lowercase
func mapToStatusAudit(row map[string]interface{}) *model.ConsentStatusAudit {
	if row == nil {
		return nil
	}

	audit := &model.ConsentStatusAudit{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["status_audit_id"].(string); ok {
		audit.StatusAuditID = id
	} else if id, ok := row["status_audit_id"].([]byte); ok {
		audit.StatusAuditID = string(id)
	}

	if consentID, ok := row["consent_id"].(string); ok {
		audit.ConsentID = consentID
	} else if consentID, ok := row["consent_id"].([]byte); ok {
		audit.ConsentID = string(consentID)
	}

	if status, ok := row["current_status"].(string); ok {
		audit.CurrentStatus = status
	} else if status, ok := row["current_status"].([]byte); ok {
		audit.CurrentStatus = string(status)
	}

	if actionTime, ok := row["action_time"].(int64); ok {
		audit.ActionTime = actionTime
	}

	if reason, ok := row["reason"].(string); ok {
		audit.Reason = &reason
	} else if reason, ok := row["reason"].([]byte); ok {
		reasonStr := string(reason)
		audit.Reason = &reasonStr
	}

	if actionBy, ok := row["action_by"].(string); ok {
		audit.ActionBy = &actionBy
	} else if actionBy, ok := row["action_by"].([]byte); ok {
		actionByStr := string(actionBy)
		audit.ActionBy = &actionByStr
	}

	if prevStatus, ok := row["previous_status"].(string); ok {
		audit.PreviousStatus = &prevStatus
	} else if prevStatus, ok := row["previous_status"].([]byte); ok {
		prevStatusStr := string(prevStatus)
		audit.PreviousStatus = &prevStatusStr
	}

	if orgID, ok := row["org_id"].(string); ok {
		audit.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		audit.OrgID = string(orgID)
	}

	return audit
}

// executeTransaction executes multiple queries within a single transaction
// This follows Thunder's functional composition pattern
func executeTransaction(dbClient provider.DBClientInterface, queries []func(tx dbmodel.TxInterface) error) error {
	tx, err := dbClient.BeginTx()
	if err != nil {
		return err
	}

	for _, query := range queries {
		if err := query(tx); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
