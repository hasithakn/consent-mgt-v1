package consentpurpose

import (
	"context"
	"fmt"

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

	QueryGetMappingsByConsentID = dbmodel.DBQuery{
		ID: "GET_MAPPINGS_BY_CONSENT_ID",
		Query: `SELECT cpm.CONSENT_ID, cpm.PURPOSE_ID, cpm.ORG_ID, cpm.VALUE, cpm.IS_USER_APPROVED, cpm.IS_MANDATORY, cp.NAME
				FROM CONSENT_PURPOSE_MAPPING cpm
				INNER JOIN CONSENT_PURPOSE cp ON cpm.PURPOSE_ID = cp.ID
				WHERE cpm.CONSENT_ID = ? AND cpm.ORG_ID = ?`,
	}

	QueryGetIDsByNames = dbmodel.DBQuery{
		ID:    "GET_IDS_BY_NAMES",
		Query: "SELECT ID, NAME FROM CONSENT_PURPOSE WHERE ORG_ID = ? AND NAME IN (%s)",
	}

	QueryDeleteMappingsByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_MAPPINGS_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_PURPOSE_MAPPING WHERE CONSENT_ID = ? AND ORG_ID = ?",
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
	GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurpose, error)
	GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurposeMapping, error)
	GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error)

	// Write operations - transactional with tx parameter
	Create(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
	Update(tx dbmodel.TxInterface, purpose *model.ConsentPurpose) error
	Delete(tx dbmodel.TxInterface, purposeID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []model.ConsentPurposeAttribute) error
	DeleteAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) error
	LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
	DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
}

// store implements the ConsentPurposeStore interface
type store struct {
	dbClient provider.DBClientInterface
}

// NewConsentPurposeStore creates a new consent purpose store
func NewConsentPurposeStore(dbClient provider.DBClientInterface) ConsentPurposeStore {
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
// Note: DBClient normalizes column names to lowercase
func mapToConsentPurpose(row map[string]interface{}) *model.ConsentPurpose {
	if row == nil {
		return nil
	}

	purpose := &model.ConsentPurpose{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["id"].(string); ok {
		purpose.ID = id
	} else if id, ok := row["id"].([]byte); ok {
		purpose.ID = string(id)
	}

	if name, ok := row["name"].(string); ok {
		purpose.Name = name
	} else if name, ok := row["name"].([]byte); ok {
		purpose.Name = string(name)
	}

	if desc, ok := row["description"].(string); ok {
		descCopy := desc
		purpose.Description = &descCopy
	} else if desc, ok := row["description"].([]byte); ok {
		descCopy := string(desc)
		purpose.Description = &descCopy
	}

	if pType, ok := row["type"].(string); ok {
		purpose.Type = pType
	} else if pType, ok := row["type"].([]byte); ok {
		purpose.Type = string(pType)
	}

	if orgID, ok := row["org_id"].(string); ok {
		purpose.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		purpose.OrgID = string(orgID)
	}

	// Initialize empty attributes map
	purpose.Attributes = make(map[string]string)

	return purpose
}

// mapToConsentPurposeAttribute maps a database row to ConsentPurposeAttribute model
// Note: DBClient normalizes column names to lowercase
func mapToConsentPurposeAttribute(row map[string]interface{}) *model.ConsentPurposeAttribute {
	if row == nil {
		return nil
	}

	attr := &model.ConsentPurposeAttribute{}

	// Handle string columns (may be string or []byte from MySQL)
	if id, ok := row["id"].(string); ok {
		attr.ID = id
	} else if id, ok := row["id"].([]byte); ok {
		attr.ID = string(id)
	}

	if purposeID, ok := row["purpose_id"].(string); ok {
		attr.PurposeID = purposeID
	} else if purposeID, ok := row["purpose_id"].([]byte); ok {
		attr.PurposeID = string(purposeID)
	}

	if key, ok := row["attr_key"].(string); ok {
		attr.Key = key
	} else if key, ok := row["attr_key"].([]byte); ok {
		attr.Key = string(key)
	}

	if value, ok := row["attr_value"].(string); ok {
		attr.Value = value
	} else if value, ok := row["attr_value"].([]byte); ok {
		attr.Value = string(value)
	}

	if orgID, ok := row["org_id"].(string); ok {
		attr.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		attr.OrgID = string(orgID)
	}

	return attr
}

// LinkPurposeToConsent links a purpose to a consent within a transaction
func (s *store) LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error {
	_, err := tx.Exec(QueryLinkPurposeToConsent.Query,
		consentID, purposeID, orgID, value, isUserApproved, isMandatory)
	return err
}

// GetPurposesByConsentID retrieves all purposes linked to a consent
func (s *store) GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurpose, error) {
	rows, err := s.dbClient.Query(QueryGetPurposesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	purposes := make([]model.ConsentPurpose, 0, len(rows))
	for _, row := range rows {
		purpose := mapToConsentPurpose(row)
		if purpose != nil {
			purposes = append(purposes, *purpose)
		}
	}

	return purposes, nil
}

// GetMappingsByConsentID retrieves all purpose mappings for a consent with their values
func (s *store) GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]model.ConsentPurposeMapping, error) {
	rows, err := s.dbClient.Query(QueryGetMappingsByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}

	mappings := make([]model.ConsentPurposeMapping, 0, len(rows))
	for _, row := range rows {
		mapping := mapToConsentPurposeMapping(row)
		if mapping != nil {
			mappings = append(mappings, *mapping)
		}
	}

	return mappings, nil
}

// GetIDsByNames retrieves purpose IDs by their names (batch lookup)
func (s *store) GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, orgID)

	for i, name := range names {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, name)
	}

	// Format query with placeholders
	query := fmt.Sprintf(QueryGetIDsByNames.Query, placeholders)

	// Create query object with formatted SQL
	formattedQuery := dbmodel.DBQuery{
		ID:    "GET_IDS_BY_NAMES_DYNAMIC",
		Query: query,
	}

	rows, err := s.dbClient.Query(formattedQuery, args...)
	if err != nil {
		return nil, err
	}

	// Build name -> ID map
	// Note: DBClient normalizes column names to lowercase
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		var id, name string

		// Handle both string and []byte types (MySQL returns []byte for strings)
		if idVal, ok := row["id"]; ok {
			if idStr, ok := idVal.(string); ok {
				id = idStr
			} else if idBytes, ok := idVal.([]byte); ok {
				id = string(idBytes)
			}
		}

		if nameVal, ok := row["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			} else if nameBytes, ok := nameVal.([]byte); ok {
				name = string(nameBytes)
			}
		}

		if id != "" && name != "" {
			result[name] = id
		}
	}
	return result, nil
}

// mapToConsentPurposeMapping maps a database row to ConsentPurposeMapping model
// Note: DBClient normalizes column names to lowercase
func mapToConsentPurposeMapping(row map[string]interface{}) *model.ConsentPurposeMapping {
	if row == nil {
		return nil
	}

	mapping := &model.ConsentPurposeMapping{}

	// Handle string columns (may be string or []byte from MySQL)
	if consentID, ok := row["consent_id"].(string); ok {
		mapping.ConsentID = consentID
	} else if consentID, ok := row["consent_id"].([]byte); ok {
		mapping.ConsentID = string(consentID)
	}

	if purposeID, ok := row["purpose_id"].(string); ok {
		mapping.PurposeID = purposeID
	} else if purposeID, ok := row["purpose_id"].([]byte); ok {
		mapping.PurposeID = string(purposeID)
	}

	if orgID, ok := row["org_id"].(string); ok {
		mapping.OrgID = orgID
	} else if orgID, ok := row["org_id"].([]byte); ok {
		mapping.OrgID = string(orgID)
	}

	if value, ok := row["value"].(string); ok {
		mapping.Value = value
	} else if value, ok := row["value"].([]byte); ok {
		mapping.Value = string(value)
	}

	// Handle boolean columns (may be bool or int64 from MySQL)
	if isUserApproved, ok := row["is_user_approved"].(bool); ok {
		mapping.IsUserApproved = isUserApproved
	} else if isUserApproved, ok := row["is_user_approved"].(int64); ok {
		mapping.IsUserApproved = isUserApproved != 0
	}

	if isMandatory, ok := row["is_mandatory"].(bool); ok {
		mapping.IsMandatory = isMandatory
	} else if isMandatory, ok := row["is_mandatory"].(int64); ok {
		mapping.IsMandatory = isMandatory != 0
	}

	if name, ok := row["name"].(string); ok {
		mapping.Name = name
	} else if name, ok := row["name"].([]byte); ok {
		mapping.Name = string(name)
	}

	return mapping
}

// DeleteMappingsByConsentID deletes all consent purpose mappings for a consent within a transaction
func (s *store) DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error {
	_, err := tx.Exec(QueryDeleteMappingsByConsentID.Query, consentID, orgID)
	return err
}
