package datasource

import (
	"context"
	"fmt"
	"strings"
)

// QueryResult represents the result of a database query
type QueryResult struct {
	// Column names
	Columns []string `json:"columns"`
	// Row data (each row is a map of column name to value)
	Rows []map[string]interface{} `json:"rows"`
	// Total number of rows returned
	RowCount int `json:"row_count"`
	// Total number of rows matching the query (before limit)
	TotalCount int `json:"total_count"`
}

// TableInfo represents metadata about a database table
type TableInfo struct {
	// Table name
	Name string `json:"name"`
	// Table description/comment
	Description string `json:"description,omitempty"`
	// Column information
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo represents metadata about a table column
type ColumnInfo struct {
	// Column name
	Name string `json:"name"`
	// Data type
	Type string `json:"type"`
	// Whether the column is nullable
	Nullable bool `json:"nullable"`
	// Column comment/description
	Description string `json:"description,omitempty"`
	// Whether the column is a primary key
	IsPrimaryKey bool `json:"is_primary_key"`
}

// DBConnector is the interface that all database connectors must implement.
// This is separate from the sync Connector interface as it focuses on
// read-only query operations rather than data synchronization.
type DBConnector interface {
	// Type returns the database type identifier (e.g., "mysql", "postgresql", "sqlite")
	Type() string

	// ValidateConnection tests the database connection with the provided config
	ValidateConnection(ctx context.Context, config map[string]interface{}) error

	// ExecuteQuery executes a read-only SQL query and returns results
	// Only SELECT statements are allowed
	ExecuteQuery(ctx context.Context, config map[string]interface{}, query string, maxRows int) (*QueryResult, error)

	// GetTableSchema returns schema information for all accessible tables
	GetTableSchema(ctx context.Context, config map[string]interface{}) ([]TableInfo, error)

	// GetTableSchemaForTable returns schema information for a specific table
	GetTableSchemaForTable(ctx context.Context, config map[string]interface{}, tableName string) (*TableInfo, error)

	// GetCreateTableSQL returns the full CREATE TABLE statement for a specific table
	// This includes column definitions, types, constraints, and table comment
	GetCreateTableSQL(ctx context.Context, config map[string]interface{}, tableName string) (string, error)

	// GetSampleData returns sample rows from a table (used for context injection)
	// Returns the column names and up to maxRows rows of data
	GetSampleData(ctx context.Context, config map[string]interface{}, tableName string, maxRows int) ([]string, []map[string]interface{}, error)

	// GetDatabaseContext returns a formatted database context string for LLM prompt injection.
	// It fetches schema, DDL, and sample data using a single database connection for efficiency.
	// The output format should match the database context specification.
	GetDatabaseContext(ctx context.Context, config map[string]interface{}, maxSampleRows int) (string, error)
}

// DBConnectorRegistry manages the registration and lookup of available database connectors
type DBConnectorRegistry struct {
	connectors map[string]DBConnector
}

// NewDBConnectorRegistry creates a new database connector registry
func NewDBConnectorRegistry() *DBConnectorRegistry {
	return &DBConnectorRegistry{
		connectors: make(map[string]DBConnector),
	}
}

// Register registers a database connector with the registry
func (r *DBConnectorRegistry) Register(connector DBConnector) error {
	if connector == nil {
		return fmt.Errorf("connector cannot be nil")
	}
	if connector.Type() == "" {
		return fmt.Errorf("connector type cannot be empty")
	}
	r.connectors[connector.Type()] = connector
	return nil
}

// Get retrieves a database connector by type
func (r *DBConnectorRegistry) Get(connectorType string) (DBConnector, error) {
	connector, exists := r.connectors[connectorType]
	if !exists {
		return nil, fmt.Errorf("database connector not found: %s", connectorType)
	}
	return connector, nil
}

// List returns all registered database connector types
func (r *DBConnectorRegistry) List() []string {
	types := make([]string, 0, len(r.connectors))
	for t := range r.connectors {
		types = append(types, t)
	}
	return types
}

// FormatFallbackCreateTable creates a simple CREATE TABLE statement from TableInfo.
// This is used as a fallback when GetCreateTableSQL fails.
func FormatFallbackCreateTable(table TableInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))
	for i, col := range table.Columns {
		nullable := "NOT NULL"
		if col.Nullable {
			nullable = "NULL"
		}
		primaryKey := ""
		if col.IsPrimaryKey {
			primaryKey = " PRIMARY KEY"
		}
		comment := ""
		if col.Description != "" {
			comment = fmt.Sprintf(" COMMENT '%s'", col.Description)
		}

		comma := ","
		if i == len(table.Columns)-1 {
			comma = ""
		}
		sb.WriteString(fmt.Sprintf("\t%s %s%s%s%s%s\n",
			col.Name, col.Type, nullable, primaryKey, comment, comma))
	}
	sb.WriteString(")ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	return sb.String()
}

// GlobalDBConnectorRegistry is the global registry for database connectors
var GlobalDBConnectorRegistry = NewDBConnectorRegistry()
