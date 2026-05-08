package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

// QueryDataSourceRepository provides data access for query data sources
type QueryDataSourceRepository struct {
	db *gorm.DB
}

// NewQueryDataSourceRepository creates a new query data source repository
func NewQueryDataSourceRepository(db *gorm.DB) interfaces.QueryDataSourceRepository {
	return &QueryDataSourceRepository{db: db}
}

// Create inserts a new query data source record
func (r *QueryDataSourceRepository) Create(ctx context.Context, ds *types.QueryDataSource) error {
	if ds == nil {
		return errors.New("query data source is nil")
	}
	if err := r.db.WithContext(ctx).Create(ds).Error; err != nil {
		return err
	}
	return nil
}

// FindByID retrieves a query data source by ID
func (r *QueryDataSourceRepository) FindByID(ctx context.Context, id string) (*types.QueryDataSource, error) {
	if id == "" {
		return nil, errors.New("id is empty")
	}
	var ds types.QueryDataSource
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(&ds).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("query data source not found")
		}
		return nil, err
	}
	return &ds, nil
}

// FindByTenant lists all query data sources for a tenant
func (r *QueryDataSourceRepository) FindByTenant(ctx context.Context, tenantID uint64) ([]*types.QueryDataSource, error) {
	var dataSources []*types.QueryDataSource
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&dataSources).Error; err != nil {
		return nil, err
	}
	return dataSources, nil
}

// Update updates an existing query data source
func (r *QueryDataSourceRepository) Update(ctx context.Context, ds *types.QueryDataSource) error {
	if ds == nil {
		return errors.New("query data source is nil")
	}
	if ds.ID == "" {
		return errors.New("query data source id is empty")
	}
	if err := r.db.WithContext(ctx).
		Model(ds).
		Updates(ds).Error; err != nil {
		return err
	}
	return nil
}

// Delete performs a soft delete
func (r *QueryDataSourceRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is empty")
	}
	if err := r.db.WithContext(ctx).
		Model(&types.QueryDataSource{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return err
	}
	return nil
}
