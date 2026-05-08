package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QueryDataSource represents a data source for AI query (sql_query tool)
type QueryDataSource struct {
	// Unique identifier
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`

	// Tenant ID for multi-tenancy
	TenantID uint64 `json:"tenant_id" gorm:"index"`

	// User-friendly name
	Name string `json:"name"`

	// Database type: mysql, postgresql, sqlite
	Type string `json:"type" gorm:"type:varchar(50);index"`

	// Encrypted configuration (host, port, username, password, etc.)
	Config JSON `json:"config" gorm:"type:jsonb"`

	// Optional description
	Description string `json:"description"`

	// Status: active, inactive, error
	Status string `json:"status" gorm:"type:varchar(32);default:'active'"`

	// Error message if status is "error"
	ErrorMessage string `json:"error_message"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Soft delete timestamp
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName specifies the table name for QueryDataSource
func (q *QueryDataSource) TableName() string {
	return "query_data_sources"
}

// BeforeCreate hook to generate UUID
func (q *QueryDataSource) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// QueryDataSourceConfig represents the unencrypted configuration structure
type QueryDataSourceConfig struct {
	// Common fields
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"`
	Charset  string `json:"charset,omitempty"`

	// SQLite specific
	FilePath string `json:"file_path,omitempty"`

	// PostgreSQL specific
	SSLMode string `json:"sslmode,omitempty"`
}

// ParseConfig parses the encrypted config JSON back to QueryDataSourceConfig
func (q *QueryDataSource) ParseConfig() (*QueryDataSourceConfig, error) {
	if len(q.Config) == 0 {
		return nil, nil
	}
	var config QueryDataSourceConfig
	if err := json.Unmarshal(q.Config, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ToJSON converts QueryDataSourceConfig to JSON
func (c *QueryDataSourceConfig) ToJSON() (JSON, error) {
	if c == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}
