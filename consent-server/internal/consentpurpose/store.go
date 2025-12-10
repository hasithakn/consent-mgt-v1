package consentpurpose

import (
	"context"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
)

// DBQuery objects for all consent purpose operations
var (
	QueryCreatePurpose = dbmodel.DBQuery{
		ID:    "CREATE_CONSENT_PURPOSE",
		Query: "INSERT INTO CONSENT_PURPOSE (ID, NAME, DESCRIPTION, TYPE, ORG_ID) VALUES (?, ?, ?, ?, ?)",
	}

	QueryGetPurposeByID = dbmodel.DBQuery{
		ID:    "GET_CONSENT_PURPOSE_BY_ID",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?",
	}

	QueryGetPurposeByName = dbmodel.DBQuery{
		ID:    "GET_CONSENT_PURPOSE_BY_NAME",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_PURPOSE WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryListPurposes = dbmodel.DBQuery{
		ID:    "LIST_CONSENT_PURPOSES",
		Query: "SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID FROM CONSENT_PURPOSE WHERE ORG_ID = ? ORDER BY NAME LIMIT ? OFFSET ?",
	}

	QueryCountPurposes = dbmodel.DBQuery{
		ID:    "COUNT_CONSENT_PURPOSES",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE ORG_ID = ?",
	}

	QueryUpdatePurpose = dbmodel.DBQuery{
		ID:    "UPDATE_CONSENT_PURPOSE",
		Query: "UPDATE CONSENT_PURPOSE SET NAME = ?, DESCRIPTION = ?, TYPE = ? WHERE ID = ? AND ORG_ID = ?",
	}

	QueryDeletePurpose = dbmodel.DBQuery{
		ID:    "DELETE_CONSENT_PURPOSE",
		Query: "DELETE FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?",
	}

	QueryCheckPurposeNameExists = dbmodel.DBQuery{
		ID:    "CHECK_PURPOSE_NAME_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryCreateAttribute = dbmodel.DBQuery{
		ID:    "CREATE_PURPOSE_ATTRIBUTE",
		Query: "INSERT INTO CONSENT_PURPOSE_ATTRIBUTE (ID, PURPOSE_ID, ATTR_KEY, ATTR_VALUE, ORG_ID) VALUES (?, ?, ?, ?, ?)",
	}

	QueryGetAttributesByPurposeID = dbmodel.DBQuery{
		ID:    "GET_ATTRIBUTES_BY_PURPOSE_ID",
		Query: "SELECT ID, PURPOSE_ID, ATTR_KEY, ATTR_VALUE, ORG_ID FROM CONSENT_PURPOSE_ATTRIBUTE WHERE PURPOSE_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAttributesByPurposeID = dbmodel.DBQuery{
		ID:    "DELETE_ATTRIBUTES_BY_PURPOSE_ID",
		Query: "DELETE FROM CONSENT_PURPOSE_ATTRIBUTE WHERE PURPOSE_ID = ? AND ORG_ID = ?",
	}

	QueryGetPurposesByConsentID = dbmodel.DBQuery{
		ID: "GET_PURPOSES_BY_CONSENT_ID",
		Query: `SELECT cp.ID, cp.NAME, cp.DESCRIPTION, cp.TYPE, cp.ORG_ID 
		        FROM CONSENT_PURPOSE cp
		        INNER JOIN CONSENT_PURPOSE_MAPPING cpm ON cp.ID = cpm.PURPOSE_ID
		        WHERE cpm.CONSENT_ID = ? AND cpm.ORG_ID = ?`,
	}

	QueryLinkPurposeToConsent = dbmodel.DBQuery{
		ID:    "LINK_PURPOSE_TO_CONSENT",
		Query: "INSERT INTO CONSENT_PURPOSE_MAPPING (CONSENT_ID, PURPOSE_ID, ORG_ID, VALUE, IS_USER_APPROVED, IS_MANDATORY) VALUES (?, ?, ?, ?, ?, ?)",
	}
)

// consentPurposeStore defines the interface for consent purpose data operations
// ConsentPurposeStore defines the interface for consent purpose data access operations
type ConsentPurposeStore interface {
	// Read operations - use dbClient directly
	GetByID(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, error)
	GetByName(ctx context.Context, name, orgID string) (*model.ConsentPurpose, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentPurpose, int, error)
	CheckNameExists(ctx context.Context, name, orgID string) (bool, error)
	GetAttributesByPurposeID(ctx context.Context, purposeID, orgID string) ([]model.ConsentPurposeAttribute, error)

	// Write operations - transactional with tx parameter
	Create(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
	Update(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
	Delete(tx dbmodel.TxInterface, purposeID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentPurposeAttribute) error
	DeleteAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) error
	LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}

// store implements the ConsentPurposeStore interface
type store struct {
	dbClient provider.DBClientInterface
}

// newConsentPurposeStore creates a new consent purpose store
func newConsentPurposeStore(dbClient provider.DBClientInterface) ConsentPurposeStore {
	return &store{
		dbClient: dbClient,
	}
}

// Create creates a new consent purpose within a transaction
func (s *store) Create(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error {
	_, err := tx.Exec(QueryCreatePurpose.Query,
		purpose.ID, purpose.Name, purpose.Description, purpose.Type, purpose.OrgID)
	return err
}

// GetByID retrieves a consent purpose by ID
func (s *store) GetByID(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, error) {
	rows, err := s.dbClient.Query(QueryGetPurposeByID, purposeID, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsentPurpose(rows[0]), nil
}

// GetByName retrieves a consent purpose by name
func (s *store) GetByName(ctx context.Context, name, orgID string) (*model.ConsentPurpose, error) {
	rows, err := s.dbClient.Query(QueryGetPurposeByName, name, orgID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return mapToConsentPurpose(rows[0]), nil
}

// List retrieves a paginated list of consent purposes
func (s *store) List(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentPurpose, int, error) {
	// Get total count
	countRows, err := s.dbClient.Query(QueryCountPurposes, orgID)
	if err != nil {
		return nil, 0, err
	}

	totalCount := 0
	if len(countRows) > 0 {
		if count, ok := countRows[0]["count"].(int64); ok {
			totalCount = int(count)
		}
	}

	// Get paginated results
	rows, err := s.dbClient.Query(QueryListPurposes, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	purposes := make([]model.ConsentPurpose, 0, len(rows))
	for _, row := range rows {
		purpose := mapToConsentPurpose(row)
		if purpose != nil {
			purposes = append(purposes, *purpose)
		}
	}

	return purposes, totalCount, nil
}

// Update updates an existing consent purpose within a transaction
func (s *store) Update(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error {
	_, err := tx.Exec(QueryUpdatePurpose.Query,
		purpose.Name, purpose.Description, purpose.Type, purpose.ID, purpose.OrgID)
	return err
}

// Delete deletes a consent purpose within a transaction
func (s *store) Delete(tx dbmodel.TxInterface, purposeID, orgID string) error {
	_, err := tx.Exec(QueryDeletePurpose.Query, purposeID, orgID)
	return err
}

// CheckNameExists checks if a purpose name already exists
func (s *store) CheckNameExists(ctx context.Context, name, orgID string) (bool, error) {
	rows, err := s.dbClient.Query(QueryCheckPurposeNameExists, name, orgID)
	if err != nil {
		return false, err
	}

	if len(rows) > 0 {
		if count, ok := rows[0]["count"].(int64); ok {
			return count > 0, nil
		}
	}
	return false, nil
}

// CreateAttributes creates multiple purpose attributes within a transaction
func (s *store) CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentPurposeAttribute) error {
	for _, attr := range attributes {
		_, err := tx.Exec(QueryCreateAttribute.Query,
			attr.ID, attr.PurposeID, attr.Key, attr.Value, attr.OrgID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAttributesByPurposeID retrieves all attributes for a purpose
func (s *store) GetAttributesByPurposeID(ctx context.Context, purposeID, orgID string) ([]model.ConsentPurposeAttribute, error) {
	rows, err := s.dbClient.Query(QueryGetAttributesByPurposeID, purposeID, orgID)
	if err != nil {
		return nil, err
	}

	attributes := make([]model.ConsentPurposeAttribute, 0, len(rows))
	for _, row := range rows {
		attr := mapToConsentPurposeAttribute(row)
		if attr != nil {
			attributes = append(attributes, *attr)
		}
	}

	return attributes, nil
}

// DeleteAttributesByPurposeID deletes all attributes for a purpose within a transaction
func (s *store) DeleteAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) error {
	_, err := tx.Exec(QueryDeleteAttributesByPurposeID.Query, purposeID, orgID)
	return err
}

// mapToConsentPurpose maps a database row to ConsentPurpose model
func mapToConsentPurpose(row map[string]interface{}) *model.ConsentPurpose {
	if row == nil {
		return nil
	}

	purpose := &model.ConsentPurpose{}

	if id, ok := row["ID"].(string); ok {
		purpose.ID = id
	}
	if name, ok := row["NAME"].(string); ok {
		purpose.Name = name
	}
	if desc, ok := row["DESCRIPTION"].(string); ok {
		descCopy := desc
		purpose.Description = &descCopy
	}
	if pType, ok := row["TYPE"].(string); ok {
		purpose.Type = pType
	}
	if orgID, ok := row["ORG_ID"].(string); ok {
		purpose.OrgID = orgID
	}

	// Initialize empty attributes map
	purpose.Attributes = make(map[string]string)

	return purpose
}

// mapToConsentPurposeAttribute maps a database row to ConsentPurposeAttribute model
func mapToConsentPurposeAttribute(row map[string]interface{}) *model.ConsentPurposeAttribute {
	if row == nil {
		return nil
	}

	attr := &model.ConsentPurposeAttribute{}

	if id, ok := row["ID"].(string); ok {
		attr.ID = id
	}
	if purposeID, ok := row["PURPOSE_ID"].(string); ok {
		attr.PurposeID = purposeID
	}
	if key, ok := row["ATTR_KEY"].(string); ok {
		attr.Key = key
	}
	if value, ok := row["ATTR_VALUE"].(string); ok {
		attr.Value = value
	}
	if orgID, ok := row["ORG_ID"].(string); ok {
		attr.OrgID = orgID
	}

	return attr
}

// LinkPurposeToConsent links a purpose to a consent within a transaction
func (s *store) LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error {
	_, err := tx.Exec(QueryLinkPurposeToConsent.Query,
		consentID, purposeID, orgID, value, isUserApproved, isMandatory)
	return err
}
