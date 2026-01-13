package consentpurposegroup

import (
	"context"
	"fmt"
	"strings"

	"github.com/wso2/consent-management-api/internal/consentpurposegroup/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/stores/interfaces"
)

// DBQuery objects for all purpose group operations
var (
	QueryCreateGroup = dbmodel.DBQuery{
		ID:    "CREATE_PURPOSE_GROUP",
		Query: "INSERT INTO CONSENT_PURPOSE_GROUP (ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetGroupByID = dbmodel.DBQuery{
		ID:    "GET_PURPOSE_GROUP_BY_ID",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE_GROUP WHERE ID = ? AND ORG_ID = ?",
	}

	QueryGetGroupByName = dbmodel.DBQuery{
		ID:    "GET_PURPOSE_GROUP_BY_NAME",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE_GROUP WHERE NAME = ? AND ORG_ID = ?",
	}

	QueryListGroups = dbmodel.DBQuery{
		ID:    "LIST_PURPOSE_GROUPS",
		Query: "SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID FROM CONSENT_PURPOSE_GROUP WHERE ORG_ID = ? ORDER BY CREATED_TIME DESC LIMIT ? OFFSET ?",
	}

	QueryCountGroups = dbmodel.DBQuery{
		ID:    "COUNT_PURPOSE_GROUPS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE_GROUP WHERE ORG_ID = ?",
	}

	QueryUpdateGroup = dbmodel.DBQuery{
		ID:    "UPDATE_PURPOSE_GROUP",
		Query: "UPDATE CONSENT_PURPOSE_GROUP SET NAME = ?, DESCRIPTION = ?, UPDATED_TIME = ? WHERE ID = ? AND ORG_ID = ?",
	}

	QueryDeleteGroup = dbmodel.DBQuery{
		ID:    "DELETE_PURPOSE_GROUP",
		Query: "DELETE FROM CONSENT_PURPOSE_GROUP WHERE ID = ? AND ORG_ID = ?",
	}

	QueryCheckGroupNameExists = dbmodel.DBQuery{
		ID:    "CHECK_GROUP_NAME_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE_GROUP WHERE NAME = ? AND CLIENT_ID = ? AND ORG_ID = ?",
	}

	QueryCheckGroupNameExistsExcluding = dbmodel.DBQuery{
		ID:    "CHECK_GROUP_NAME_EXISTS_EXCLUDING",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE_GROUP WHERE NAME = ? AND CLIENT_ID = ? AND ORG_ID = ? AND ID != ?",
	}

	QueryLinkPurposeToGroup = dbmodel.DBQuery{
		ID:    "LINK_PURPOSE_TO_GROUP",
		Query: "INSERT INTO CONSENT_PURPOSE_GROUP_MAPPING (GROUP_ID, PURPOSE_ID, IS_MANDATORY, ORG_ID) VALUES (?, ?, ?, ?)",
	}

	QueryGetGroupPurposes = dbmodel.DBQuery{
		ID: "GET_GROUP_PURPOSES",
		Query: `SELECT m.PURPOSE_ID, p.NAME as PURPOSE_NAME, m.IS_MANDATORY 
				FROM CONSENT_PURPOSE_GROUP_MAPPING m 
				JOIN CONSENT_PURPOSE p ON m.PURPOSE_ID = p.ID AND m.ORG_ID = p.ORG_ID 
				WHERE m.GROUP_ID = ? AND m.ORG_ID = ?`,
	}

	QueryDeleteGroupPurposes = dbmodel.DBQuery{
		ID:    "DELETE_GROUP_PURPOSES",
		Query: "DELETE FROM CONSENT_PURPOSE_GROUP_MAPPING WHERE GROUP_ID = ? AND ORG_ID = ?",
	}

	QueryGetPurposeIDByName = dbmodel.DBQuery{
		ID:    "GET_PURPOSE_ID_BY_NAME",
		Query: "SELECT ID FROM CONSENT_PURPOSE WHERE NAME = ? AND ORG_ID = ?",
	}

	queryCheckPurposeInGroups = dbmodel.DBQuery{
		ID:    "CHECK_PURPOSE_IN_GROUPS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_PURPOSE_GROUP_MAPPING WHERE PURPOSE_ID = ? AND ORG_ID = ?",
	}
)

// store implements the ConsentPurposeGroupStore interface
type store struct {
	dbClient provider.DBClientInterface
}

// NewPurposeGroupStore creates a new purpose group store
func NewPurposeGroupStore(dbClient provider.DBClientInterface) interfaces.ConsentPurposeGroupStore {
	return &store{
		dbClient: dbClient,
	}
}

// CreateGroup creates a new purpose group
func (s *store) CreateGroup(tx dbmodel.TxInterface, group *model.PurposeGroup) error {
	_, err := tx.Exec(QueryCreateGroup.Query,
		group.ID,
		group.Name,
		group.Description,
		group.ClientID,
		group.CreatedTime,
		group.UpdatedTime,
		group.OrgID,
	)
	return err
}

// GetGroupByID retrieves a purpose group by ID with its purposes
func (s *store) GetGroupByID(ctx context.Context, groupID, orgID string) (*model.PurposeGroup, error) {
	var group model.PurposeGroup
	rows, err := s.dbClient.Query(QueryGetGroupByID, groupID, orgID)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("purpose group not found")
	}

	// Extract values from the first row
	row := rows[0]
	if id, ok := row["id"].([]uint8); ok {
		group.ID = string(id)
	}
	if name, ok := row["name"].([]uint8); ok {
		group.Name = string(name)
	}
	if desc, ok := row["description"]; ok && desc != nil {
		if descBytes, ok := desc.([]uint8); ok {
			descStr := string(descBytes)
			group.Description = &descStr
		}
	}
	if clientID, ok := row["client_id"].([]uint8); ok {
		group.ClientID = string(clientID)
	}
	if createdTime, ok := row["created_time"].(int64); ok {
		group.CreatedTime = createdTime
	}
	if updatedTime, ok := row["updated_time"].(int64); ok {
		group.UpdatedTime = updatedTime
	}
	if orgID, ok := row["org_id"].([]uint8); ok {
		group.OrgID = string(orgID)
	}

	// Load purposes for the group
	purposes, err := s.GetGroupPurposes(ctx, groupID, orgID)
	if err != nil {
		return nil, err
	}
	group.Purposes = purposes

	return &group, nil
}

// GetGroupByName retrieves a purpose group by name
func (s *store) GetGroupByName(ctx context.Context, name, orgID string) (*model.PurposeGroup, error) {
	var group model.PurposeGroup
	rows, err := s.dbClient.Query(QueryGetGroupByName, name, orgID)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("purpose group not found")
	}

	// Extract values from the first row
	row := rows[0]
	if id, ok := row["id"].([]uint8); ok {
		group.ID = string(id)
	}
	if name, ok := row["name"].([]uint8); ok {
		group.Name = string(name)
	}
	if desc, ok := row["description"]; ok && desc != nil {
		if descBytes, ok := desc.([]uint8); ok {
			descStr := string(descBytes)
			group.Description = &descStr
		}
	}
	if clientID, ok := row["client_id"].([]uint8); ok {
		group.ClientID = string(clientID)
	}
	if createdTime, ok := row["created_time"].(int64); ok {
		group.CreatedTime = createdTime
	}
	if updatedTime, ok := row["updated_time"].(int64); ok {
		group.UpdatedTime = updatedTime
	}
	if orgIDVal, ok := row["org_id"].([]uint8); ok {
		group.OrgID = string(orgIDVal)
	}

	// Load purposes for the group
	purposes, err := s.GetGroupPurposes(ctx, group.ID, orgID)
	if err != nil {
		return nil, err
	}
	group.Purposes = purposes

	return &group, nil
}

// ListGroups retrieves a list of purpose groups with filtering
func (s *store) ListGroups(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.PurposeGroup, int, error) {
	// Build dynamic query based on filters
	query := `SELECT ID, NAME, DESCRIPTION, CLIENT_ID, CREATED_TIME, UPDATED_TIME, ORG_ID 
			  FROM CONSENT_PURPOSE_GROUP 
			  WHERE ORG_ID = ?`
	countQuery := `SELECT COUNT(*) as count FROM CONSENT_PURPOSE_GROUP WHERE ORG_ID = ?`

	args := []interface{}{orgID}
	countArgs := []interface{}{orgID}

	// Filter by name (exact match or partial match with LIKE)
	if name != "" {
		query += ` AND NAME = ?`
		countQuery += ` AND NAME = ?`
		args = append(args, name)
		countArgs = append(countArgs, name)
	}

	// Filter by clientIDs
	if len(clientIDs) > 0 {
		placeholders := strings.Repeat("?,", len(clientIDs))
		placeholders = placeholders[:len(placeholders)-1]
		query += ` AND CLIENT_ID IN (` + placeholders + `)`
		countQuery += ` AND CLIENT_ID IN (` + placeholders + `)`
		for _, clientID := range clientIDs {
			args = append(args, clientID)
			countArgs = append(countArgs, clientID)
		}
	}

	// Filter by purposeNames - AND logic: group must contain ALL specified purposes
	if len(purposeNames) > 0 {
		// For each purpose name, ensure the group contains it
		for _, purposeName := range purposeNames {
			subQuery := ` AND EXISTS (
				SELECT 1 FROM CONSENT_PURPOSE_GROUP_MAPPING m
				JOIN CONSENT_PURPOSE p ON m.PURPOSE_ID = p.ID AND m.ORG_ID = p.ORG_ID
				WHERE m.GROUP_ID = CONSENT_PURPOSE_GROUP.ID 
				  AND m.ORG_ID = CONSENT_PURPOSE_GROUP.ORG_ID
				  AND p.NAME = ?
			)`
			query += subQuery
			countQuery += subQuery
			args = append(args, purposeName)
			countArgs = append(countArgs, purposeName)
		}
	}

	// Get total count
	var total int
	countQueryDB := dbmodel.DBQuery{
		ID:    "COUNT_FILTERED_GROUPS",
		Query: countQuery,
	}
	rows, err := s.dbClient.Query(countQueryDB, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	if len(rows) > 0 {
		if count, ok := rows[0]["count"].(int64); ok {
			total = int(count)
		}
	}

	// Add sorting and pagination
	query += ` ORDER BY CREATED_TIME DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	// Execute query
	var groups []model.PurposeGroup
	listQueryDB := dbmodel.DBQuery{
		ID:    "LIST_FILTERED_GROUPS",
		Query: query,
	}
	rows, err = s.dbClient.Query(listQueryDB, args...)
	if err != nil {
		return nil, 0, err
	}

	for _, row := range rows {
		var group model.PurposeGroup
		if id, ok := row["id"].([]uint8); ok {
			group.ID = string(id)
		}
		if name, ok := row["name"].([]uint8); ok {
			group.Name = string(name)
		}
		if desc, ok := row["description"]; ok && desc != nil {
			if descBytes, ok := desc.([]uint8); ok {
				descStr := string(descBytes)
				group.Description = &descStr
			}
		}
		if clientID, ok := row["client_id"].([]uint8); ok {
			group.ClientID = string(clientID)
		}
		if createdTime, ok := row["created_time"].(int64); ok {
			group.CreatedTime = createdTime
		}
		if updatedTime, ok := row["updated_time"].(int64); ok {
			group.UpdatedTime = updatedTime
		}
		if orgID, ok := row["org_id"].([]uint8); ok {
			group.OrgID = string(orgID)
		}
		groups = append(groups, group)
	}

	// Load purposes for each group
	for i := range groups {
		purposes, err := s.GetGroupPurposes(ctx, groups[i].ID, orgID)
		if err != nil {
			return nil, 0, err
		}
		groups[i].Purposes = purposes
	}

	return groups, total, nil
}

// UpdateGroup updates an existing purpose group
func (s *store) UpdateGroup(tx dbmodel.TxInterface, group *model.PurposeGroup) error {
	_, err := tx.Exec(QueryUpdateGroup.Query,
		group.Name,
		group.Description,
		group.UpdatedTime,
		group.ID,
		group.OrgID,
	)
	return err
}

// DeleteGroup deletes a purpose group
func (s *store) DeleteGroup(tx dbmodel.TxInterface, groupID, orgID string) error {
	_, err := tx.Exec(QueryDeleteGroup.Query, groupID, orgID)
	return err
}

// CheckGroupNameExists checks if a group name exists for a client
func (s *store) CheckGroupNameExists(ctx context.Context, name, clientID, orgID string, excludeGroupID *string) (bool, error) {
	var count int
	var rows []map[string]interface{}
	var err error

	if excludeGroupID != nil {
		rows, err = s.dbClient.Query(QueryCheckGroupNameExistsExcluding, name, clientID, orgID, *excludeGroupID)
	} else {
		rows, err = s.dbClient.Query(QueryCheckGroupNameExists, name, clientID, orgID)
	}

	if err != nil {
		return false, err
	}

	if len(rows) > 0 {
		if countVal, ok := rows[0]["count"].(int64); ok {
			count = int(countVal)
		}
	}

	return count > 0, nil
}

// LinkPurposeToGroup links a purpose to a group
func (s *store) LinkPurposeToGroup(tx dbmodel.TxInterface, groupID, purposeID, orgID string, isMandatory bool) error {
	_, err := tx.Exec(QueryLinkPurposeToGroup.Query,
		groupID,
		purposeID,
		isMandatory,
		orgID,
	)
	return err
}

// GetGroupPurposes retrieves all purposes for a group
func (s *store) GetGroupPurposes(ctx context.Context, groupID, orgID string) ([]model.PurposeGroupPurpose, error) {
	rows, err := s.dbClient.Query(QueryGetGroupPurposes, groupID, orgID)
	if err != nil {
		return nil, err
	}

	var purposes []model.PurposeGroupPurpose
	for _, row := range rows {
		var p model.PurposeGroupPurpose
		if purposeID, ok := row["purpose_id"].([]uint8); ok {
			p.PurposeID = string(purposeID)
		}
		if purposeName, ok := row["purpose_name"].([]uint8); ok {
			p.PurposeName = string(purposeName)
		}
		if isMandatory, ok := row["is_mandatory"].(int64); ok {
			p.IsMandatory = isMandatory != 0
		}
		purposes = append(purposes, p)
	}
	return purposes, nil
}

// DeleteGroupPurposes deletes all purpose mappings for a group
func (s *store) DeleteGroupPurposes(tx dbmodel.TxInterface, groupID, orgID string) error {
	_, err := tx.Exec(QueryDeleteGroupPurposes.Query, groupID, orgID)
	return err
}

// GetPurposeIDByName retrieves a purpose ID by name
func (s *store) GetPurposeIDByName(ctx context.Context, purposeName, orgID string) (string, error) {
	var purposeID string
	rows, err := s.dbClient.Query(QueryGetPurposeIDByName, purposeName, orgID)
	if err != nil {
		return "", err
	}

	if len(rows) == 0 {
		return "", fmt.Errorf("purpose '%s' not found", purposeName)
	}

	if id, ok := rows[0]["id"].([]uint8); ok {
		purposeID = string(id)
	}
	return purposeID, nil
}

// ValidatePurposeNames validates that all purpose names exist and returns a map of name -> ID
func (s *store) ValidatePurposeNames(ctx context.Context, purposeNames []string, orgID string) (map[string]string, error) {
	if len(purposeNames) == 0 {
		return map[string]string{}, nil
	}

	placeholders := strings.Repeat("?,", len(purposeNames))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf("SELECT NAME, ID FROM CONSENT_PURPOSE WHERE ORG_ID = ? AND NAME IN (%s)", placeholders)

	args := []interface{}{orgID}
	for _, name := range purposeNames {
		args = append(args, name)
	}

	validateQueryDB := dbmodel.DBQuery{
		ID:    "VALIDATE_PURPOSE_NAMES",
		Query: query,
	}
	rows, err := s.dbClient.Query(validateQueryDB, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, row := range rows {
		var name, id string
		if nameBytes, ok := row["name"].([]uint8); ok {
			name = string(nameBytes)
		}
		if idBytes, ok := row["id"].([]uint8); ok {
			id = string(idBytes)
		}
		result[name] = id
	}

	return result, nil
}

// IsPurposeUsedInGroups checks if a purpose is used in any purpose group
func (s *store) IsPurposeUsedInGroups(ctx context.Context, purposeID, orgID string) (bool, error) {

	rows, err := s.dbClient.Query(queryCheckPurposeInGroups, purposeID, orgID)
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
