package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/datasource"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// QueryDataSourceService implements the QueryDataSourceService interface
type QueryDataSourceService struct {
	repo interfaces.QueryDataSourceRepository
}

// NewQueryDataSourceService creates a new query data source service
func NewQueryDataSourceService(repo interfaces.QueryDataSourceRepository) interfaces.QueryDataSourceService {
	return &QueryDataSourceService{repo: repo}
}

// Create creates a new query data source
func (s *QueryDataSourceService) Create(ctx context.Context, ds *types.QueryDataSource) (*types.QueryDataSource, error) {
	if ds == nil {
		return nil, fmt.Errorf("query data source is nil")
	}

	// Validate the connection before creating
	if err := s.ValidateConnection(ctx, ds); err != nil {
		return nil, fmt.Errorf("connection validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, ds); err != nil {
		return nil, err
	}

	logger.Infof(ctx, "Query data source created: id=%s, name=%s, type=%s", ds.ID, ds.Name, ds.Type)
	return ds, nil
}

// GetByID retrieves a query data source by ID
func (s *QueryDataSourceService) GetByID(ctx context.Context, id string) (*types.QueryDataSource, error) {
	return s.repo.FindByID(ctx, id)
}

// ListByTenant lists all query data sources for a tenant
func (s *QueryDataSourceService) ListByTenant(ctx context.Context, tenantID uint64) ([]*types.QueryDataSource, error) {
	return s.repo.FindByTenant(ctx, tenantID)
}

// Update updates an existing query data source
func (s *QueryDataSourceService) Update(ctx context.Context, ds *types.QueryDataSource) (*types.QueryDataSource, error) {
	if ds == nil {
		return nil, fmt.Errorf("query data source is nil")
	}

	// Validate the connection before updating
	if err := s.ValidateConnection(ctx, ds); err != nil {
		return nil, fmt.Errorf("connection validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, ds); err != nil {
		return nil, err
	}

	logger.Infof(ctx, "Query data source updated: id=%s, name=%s", ds.ID, ds.Name)
	return ds, nil
}

// Delete soft deletes a query data source
func (s *QueryDataSourceService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	logger.Infof(ctx, "Query data source deleted: id=%s", id)
	return nil
}

// ValidateConnection tests the connection to a database
func (s *QueryDataSourceService) ValidateConnection(ctx context.Context, ds *types.QueryDataSource) error {
	if ds == nil {
		return fmt.Errorf("query data source is nil")
	}

	// Parse config
	config, err := ds.ParseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get the database connector
	connector, err := datasource.GlobalDBConnectorRegistry.Get(ds.Type)
	if err != nil {
		return fmt.Errorf("unsupported database type: %s", ds.Type)
	}

	// Convert config to map
	configMap := make(map[string]interface{})
	if config.Host != "" {
		configMap["host"] = config.Host
	}
	if config.Port > 0 {
		configMap["port"] = config.Port
	}
	if config.Username != "" {
		configMap["username"] = config.Username
	}
	if config.Password != "" {
		configMap["password"] = config.Password
	}
	if config.Database != "" {
		configMap["database"] = config.Database
	}
	if config.Charset != "" {
		configMap["charset"] = config.Charset
	}
	if config.FilePath != "" {
		configMap["file_path"] = config.FilePath
	}
	if config.SSLMode != "" {
		configMap["sslmode"] = config.SSLMode
	}

	// Test connection
	if err := connector.ValidateConnection(ctx, configMap); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}
