package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// QueryDataSourceService defines the interface for AI query data source management
type QueryDataSourceService interface {
	// Create creates a new query data source
	Create(ctx context.Context, ds *types.QueryDataSource) (*types.QueryDataSource, error)

	// GetByID retrieves a query data source by ID
	GetByID(ctx context.Context, id string) (*types.QueryDataSource, error)

	// ListByTenant lists all query data sources for a tenant
	ListByTenant(ctx context.Context, tenantID uint64) ([]*types.QueryDataSource, error)

	// Update updates an existing query data source
	Update(ctx context.Context, ds *types.QueryDataSource) (*types.QueryDataSource, error)

	// Delete soft deletes a query data source
	Delete(ctx context.Context, id string) error

	// ValidateConnection tests the connection to a database
	ValidateConnection(ctx context.Context, ds *types.QueryDataSource) error
}

// QueryDataSourceRepository defines database access patterns for query data sources
type QueryDataSourceRepository interface {
	// Create inserts a new query data source record
	Create(ctx context.Context, ds *types.QueryDataSource) error

	// FindByID retrieves a query data source by ID
	FindByID(ctx context.Context, id string) (*types.QueryDataSource, error)

	// FindByTenant lists all query data sources for a tenant
	FindByTenant(ctx context.Context, tenantID uint64) ([]*types.QueryDataSource, error)

	// Update updates an existing query data source
	Update(ctx context.Context, ds *types.QueryDataSource) error

	// Delete performs a soft delete
	Delete(ctx context.Context, id string) error
}
