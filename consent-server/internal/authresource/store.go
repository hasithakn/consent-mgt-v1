package authresource

import (
	"context"
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
)

// DBQuery objects for all auth resource operations
var (
	QueryCreateAuthResource = dbmodel.DBQuery{
		ID:    "CREATE_AUTH_RESOURCE",
		Query: "INSERT INTO CONSENT_AUTH_RESOURCE (AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	}

	QueryGetAuthResourceByID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCE_BY_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryGetAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCES_BY_CONSENT_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAuthResource = dbmodel.DBQuery{
		ID:    "UPDATE_AUTH_RESOURCE",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, USER_ID = ?, RESOURCES = ?, UPDATED_TIME = ? WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAuthResourceStatus = dbmodel.DBQuery{
		ID:    "UPDATE_AUTH_RESOURCE_STATUS",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, UPDATED_TIME = ? WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAuthResource = dbmodel.DBQuery{
		ID:    "DELETE_AUTH_RESOURCE",
		Query: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryDeleteAuthResourcesByConsentID = dbmodel.DBQuery{
		ID:    "DELETE_AUTH_RESOURCES_BY_CONSENT_ID",
		Query: "DELETE FROM CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}

	QueryCheckAuthResourceExists = dbmodel.DBQuery{
		ID:    "CHECK_AUTH_RESOURCE_EXISTS",
		Query: "SELECT COUNT(*) as count FROM CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?",
	}

	QueryGetAuthResourcesByUserID = dbmodel.DBQuery{
		ID:    "GET_AUTH_RESOURCES_BY_USER_ID",
		Query: "SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS, UPDATED_TIME, RESOURCES, ORG_ID FROM CONSENT_AUTH_RESOURCE WHERE USER_ID = ? AND ORG_ID = ?",
	}

	QueryUpdateAllStatusByConsentID = dbmodel.DBQuery{
		ID:    "UPDATE_ALL_STATUS_BY_CONSENT_ID",
		Query: "UPDATE CONSENT_AUTH_RESOURCE SET AUTH_STATUS = ?, UPDATED_TIME = ? WHERE CONSENT_ID = ? AND ORG_ID = ?",
	}
)

// authResourceStore defines the interface for auth resource data operations
type authResourceStore interface {
	Create(ctx context.Context, authResource *model.AuthResource) error
	GetByID(ctx context.Context, authID, orgID string) (*model.AuthResource, error)
	GetByConsentID(ctx context.Context, consentID, orgID string) ([]model.AuthResource, error)
	Update(ctx context.Context, authResource *model.AuthResource) error
	UpdateStatus(ctx context.Context, authID, orgID, status string, updatedTime int64) error
	Delete(ctx context.Context, authID, orgID string) error
	DeleteByConsentID(ctx context.Context, consentID, orgID string) error
	Exists(ctx context.Context, authID, orgID string) (bool, error)
	GetByUserID(ctx context.Context, userID, orgID string) ([]model.AuthResource, error)
	UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID, status string, updatedTime int64) error
}

// store implements authResourceStore using Thunder pattern
type store struct {
	dbClient provider.DBClientInterface
}

// newAuthResourceStore creates a new auth resource store
func newAuthResourceStore(dbClient provider.DBClientInterface) authResourceStore {
	return &store{
		dbClient: dbClient,
	}
}

// Create creates a new auth resource
func (s *store) Create(ctx context.Context, authResource *model.AuthResource) error {
	_, err := s.dbClient.Execute(QueryCreateAuthResource,
		authResource.AuthID,
		authResource.ConsentID,
		authResource.AuthType,
		authResource.UserID,
		authResource.AuthStatus,
		authResource.UpdatedTime,
		authResource.Resources,
		authResource.OrgID,
	)
	return err
}

// GetByID retrieves an auth resource by ID
func (s *store) GetByID(ctx context.Context, authID, orgID string) (*model.AuthResource, error) {
	results, err := s.dbClient.Query(QueryGetAuthResourceByID, authID, orgID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("auth resource not found")
	}
	return mapToAuthResource(results[0]), nil
}

// GetByConsentID retrieves all auth resources for a consent
func (s *store) GetByConsentID(ctx context.Context, consentID, orgID string) ([]model.AuthResource, error) {
	results, err := s.dbClient.Query(QueryGetAuthResourcesByConsentID, consentID, orgID)
	if err != nil {
		return nil, err
	}
	
	authResources := make([]model.AuthResource, 0, len(results))
	for _, row := range results {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// Update updates an auth resource
func (s *store) Update(ctx context.Context, authResource *model.AuthResource) error {
	_, err := s.dbClient.Execute(QueryUpdateAuthResource,
		authResource.AuthStatus,
		authResource.UserID,
		authResource.Resources,
		authResource.UpdatedTime,
		authResource.AuthID,
		authResource.OrgID,
	)
	return err
}

// UpdateStatus updates only the status of an auth resource
func (s *store) UpdateStatus(ctx context.Context, authID, orgID, status string, updatedTime int64) error {
	_, err := s.dbClient.Execute(QueryUpdateAuthResourceStatus, status, updatedTime, authID, orgID)
	return err
}

// Delete deletes an auth resource
func (s *store) Delete(ctx context.Context, authID, orgID string) error {
	_, err := s.dbClient.Execute(QueryDeleteAuthResource, authID, orgID)
	return err
}

// DeleteByConsentID deletes all auth resources for a consent
func (s *store) DeleteByConsentID(ctx context.Context, consentID, orgID string) error {
	_, err := s.dbClient.Execute(QueryDeleteAuthResourcesByConsentID, consentID, orgID)
	return err
}

// Exists checks if an auth resource exists
func (s *store) Exists(ctx context.Context, authID, orgID string) (bool, error) {
	results, err := s.dbClient.Query(QueryCheckAuthResourceExists, authID, orgID)
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, nil
	}
	count, ok := results[0]["count"].(int64)
	if !ok {
		return false, nil
	}
	return count > 0, nil
}

// GetByUserID retrieves all auth resources for a user
func (s *store) GetByUserID(ctx context.Context, userID, orgID string) ([]model.AuthResource, error) {
	results, err := s.dbClient.Query(QueryGetAuthResourcesByUserID, userID, orgID)
	if err != nil {
		return nil, err
	}
	
	authResources := make([]model.AuthResource, 0, len(results))
	for _, row := range results {
		authResources = append(authResources, *mapToAuthResource(row))
	}
	return authResources, nil
}

// UpdateAllStatusByConsentID updates status for all auth resources of a consent
func (s *store) UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID, status string, updatedTime int64) error {
	_, err := s.dbClient.Execute(QueryUpdateAllStatusByConsentID, status, updatedTime, consentID, orgID)
	return err
}

// mapToAuthResource converts a database row map to AuthResource
func mapToAuthResource(row map[string]interface{}) *model.AuthResource {
	authResource := &model.AuthResource{}
	
	if v, ok := row["AUTH_ID"].(string); ok {
		authResource.AuthID = v
	}
	if v, ok := row["CONSENT_ID"].(string); ok {
		authResource.ConsentID = v
	}
	if v, ok := row["AUTH_TYPE"].(string); ok {
		authResource.AuthType = v
	}
	if v, ok := row["USER_ID"].(string); ok {
		authResource.UserID = &v
	}
	if v, ok := row["AUTH_STATUS"].(string); ok {
		authResource.AuthStatus = v
	}
	if v, ok := row["UPDATED_TIME"].(int64); ok {
		authResource.UpdatedTime = v
	}
	if v, ok := row["RESOURCES"].(string); ok {
		authResource.Resources = &v
	}
	if v, ok := row["ORG_ID"].(string); ok {
		authResource.OrgID = v
	}
	
	return authResource
}

// executeTransaction is a helper for functional transaction composition
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
