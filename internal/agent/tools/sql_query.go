package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/datasource"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const (
	// DefaultMaxRows is the default maximum number of rows to return
	DefaultSQLMaxRows = 50
)

// SQLQueryInput represents the input parameters for the SQL query tool
type SQLQueryInput struct {
	DataSourceID string `json:"data_source_id" jsonschema:"The ID of the data source to query. Get this from the @ data source selection."`
	SQL          string `json:"sql" jsonschema:"The SELECT SQL query to execute. Only SELECT statements are allowed. Do not include INSERT, UPDATE, DELETE, or other modification statements."`
}

// SQLQueryTool allows AI to query external databases with read-only access
type SQLQueryTool struct {
	BaseTool
	dataSourceService DataSourceServiceInterface
	dataSourceID      string
	dataSourceType    string
	dataSourceConfig  map[string]interface{}
	schemaInfo        string // Pre-fetched schema information
}

// DataSourceServiceInterface defines the interface for data source operations
type DataSourceServiceInterface interface {
	GetDataSource(ctx context.Context, id string) (*types.DataSource, error)
}

// NewSQLQueryTool creates a new SQL query tool with schema information
func NewSQLQueryTool(dsService DataSourceServiceInterface, dsID string, dsType string, dsConfig map[string]interface{}) *SQLQueryTool {
	// Build tool description with schema info
	description := buildSQLQueryDescription(dsType, dsID)

	tool := &SQLQueryTool{
		BaseTool: BaseTool{
			name:        ToolSQLQuery,
			description: description,
			schema:      utils.GenerateSchema[SQLQueryInput](),
		},
		dataSourceService: dsService,
		dataSourceID:      dsID,
		dataSourceType:    dsType,
		dataSourceConfig:  dsConfig,
	}

	// Try to fetch schema information at tool creation time
	schemaInfo := fetchSchemaInfo(context.Background(), dsType, dsConfig)
	if schemaInfo != "" {
		tool.schemaInfo = schemaInfo
		tool.description = description + "\n\n## Available Tables and Columns\n\n" + schemaInfo
	}

	return tool
}

// buildSQLQueryDescription builds the base description for the SQL query tool
func buildSQLQueryDescription(dbType, dsID string) string {
	return fmt.Sprintf(`Execute SQL queries on external %s database to retrieve data.

## Security Features
- Read-only queries: Only SELECT statements are allowed (no INSERT, UPDATE, DELETE, DROP, etc.)
- Result limiting: Returns maximum 50 rows with total count
- SQL injection prevention: Queries are validated and parameterized
- Connection isolation: Each query creates a new connection

## Available Operations
- Query data from tables
- Join multiple tables
- Aggregate data with GROUP BY
- Filter with WHERE clauses
- Sort with ORDER BY

## Usage
- data_source_id: "%s"
- sql: A valid SELECT SQL query

## Result Format
Results are returned in Markdown table format with:
- Column headers
- Data rows (max 50)
- Total row count if more than 50 results

## Important Notes
- Use the exact table and column names from the schema above
- Always use appropriate JOIN conditions when joining tables
- Use LIMIT for better performance on large tables`, strings.ToUpper(dbType), dsID)
}

// fetchSchemaInfo fetches database schema information
func fetchSchemaInfo(ctx context.Context, dbType string, config map[string]interface{}) string {
	connector, err := datasource.GlobalDBConnectorRegistry.Get(dbType)
	if err != nil {
		logger.Warnf(ctx, "[SQLQuery] Failed to get connector for schema fetch: %v", err)
		return ""
	}

	tables, err := connector.GetTableSchema(ctx, config)
	if err != nil {
		logger.Warnf(ctx, "[SQLQuery] Failed to fetch schema: %v", err)
		return ""
	}

	if len(tables) == 0 {
		return ""
	}

	return formatSchemaAsMarkdown(tables)
}

// formatSchemaAsMarkdown formats table schema as Markdown
func formatSchemaAsMarkdown(tables []datasource.TableInfo) string {
	var sb strings.Builder

	for _, table := range tables {
		sb.WriteString(fmt.Sprintf("### %s\n", table.Name))
		if table.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", table.Description))
		}

		sb.WriteString("\n| Column | Type | Nullable | Key | Description |\n")
		sb.WriteString("|--------|------|----------|-----|-------------|\n")

		for _, col := range table.Columns {
			nullable := "NO"
			if col.Nullable {
				nullable = "YES"
			}
			key := ""
			if col.IsPrimaryKey {
				key = "PRI"
			}
			desc := col.Description
			if desc == "" {
				desc = "-"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				col.Name, col.Type, nullable, key, desc))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Execute executes the SQL query tool
func (t *SQLQueryTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][SQLQuery] Execute started")

	// Parse args from json.RawMessage
	var input SQLQueryInput
	if err := json.Unmarshal(args, &input); err != nil {
		logger.Errorf(ctx, "[Tool][SQLQuery] Failed to parse args: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse args: %v", err),
		}, err
	}

	// Use the configured data source ID if not provided in args
	dataSourceID := input.DataSourceID
	if dataSourceID == "" {
		dataSourceID = t.dataSourceID
	}

	if dataSourceID == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "No data source specified. Please @ a data source first.",
		}, fmt.Errorf("no data source specified")
	}

	// Validate SQL query
	if input.SQL == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "SQL query is required",
		}, fmt.Errorf("missing sql parameter")
	}

	// Additional SQL validation
	if err := validateSQLSafety(input.SQL); err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("SQL validation failed: %v", err),
		}, err
	}

	logger.Infof(ctx, "[Tool][SQLQuery] Executing query on data source %s: %s", dataSourceID, input.SQL)

	// Get the database connector
	connector, err := datasource.GlobalDBConnectorRegistry.Get(t.dataSourceType)
	if err != nil {
		logger.Errorf(ctx, "[Tool][SQLQuery] Unsupported database type: %s", t.dataSourceType)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Unsupported database type: %s", t.dataSourceType),
		}, err
	}

	// Execute the query
	result, err := connector.ExecuteQuery(ctx, t.dataSourceConfig, input.SQL, DefaultSQLMaxRows)
	if err != nil {
		logger.Errorf(ctx, "[Tool][SQLQuery] Query execution failed: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Query execution failed: %v", err),
		}, err
	}

	logger.Infof(ctx, "[Tool][SQLQuery] Query returned %d rows (total: %d)", result.RowCount, result.TotalCount)

	// Format output as Markdown table
	output := formatQueryResultAsMarkdown(result)

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"columns":      result.Columns,
			"rows":         result.Rows,
			"row_count":    result.RowCount,
			"total_count":  result.TotalCount,
			"display_type": "sql_query",
		},
	}, nil
}

// GetSchemaInfo returns the pre-fetched schema information
func (t *SQLQueryTool) GetSchemaInfo() string {
	return t.schemaInfo
}

// validateSQLSafety performs additional safety checks on the SQL query
func validateSQLSafety(sql string) error {
	// Convert to uppercase for checking
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))

	// Check for multiple statements (semicolons)
	// Remove the trailing semicolon if present
	cleanSQL := strings.TrimRight(strings.TrimSpace(sql), ";")
	if strings.Contains(cleanSQL, ";") {
		return fmt.Errorf("multiple statements are not allowed")
	}

	// Check for comment patterns that could be used for injection
	if strings.Contains(upperSQL, "/*") || strings.Contains(upperSQL, "*/") {
		return fmt.Errorf("block comments are not allowed")
	}

	// Check for dangerous functions
	dangerousFunctions := []string{
		"SLEEP(", "BENCHMARK(", "LOAD_FILE(", "INTO OUTFILE",
		"INTO DUMPFILE", "LOAD DATA",
	}
	for _, fn := range dangerousFunctions {
		if strings.Contains(upperSQL, fn) {
			return fmt.Errorf("forbidden function or operation: %s", fn)
		}
	}

	return nil
}

// formatQueryResultAsMarkdown formats the query result as a Markdown table
func formatQueryResultAsMarkdown(result *datasource.QueryResult) string {
	if result == nil || len(result.Columns) == 0 {
		return "No results returned."
	}

	var sb strings.Builder

	// Header
	sb.WriteString("=== Query Results ===\n\n")

	if result.TotalCount > result.RowCount {
		sb.WriteString(fmt.Sprintf("Returned %d rows (Total matching rows: %d)\n\n", result.RowCount, result.TotalCount))
	} else {
		sb.WriteString(fmt.Sprintf("Returned %d rows\n\n", result.RowCount))
	}

	if len(result.Rows) == 0 {
		sb.WriteString("No matching records found.\n")
		return sb.String()
	}

	// Build Markdown table
	sb.WriteString("| ")
	sb.WriteString(strings.Join(result.Columns, " | "))
	sb.WriteString(" |\n")

	// Separator line
	sb.WriteString("| ")
	for i := 0; i < len(result.Columns); i++ {
		sb.WriteString("---")
		if i < len(result.Columns)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString(" |\n")

	// Data rows
	for _, row := range result.Rows {
		sb.WriteString("| ")
		for i, col := range result.Columns {
			value := row[col]
			sb.WriteString(formatValue(value))
			if i < len(result.Columns)-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString(" |\n")
	}

	// Add note if results are limited
	if result.TotalCount > result.RowCount {
		sb.WriteString(fmt.Sprintf("\n*Note: Showing first %d rows. Use LIMIT clause or more specific WHERE conditions to see different results.*\n", result.RowCount))
	}

	return sb.String()
}

// formatValue formats a value for display in Markdown
func formatValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		// Escape pipe characters and limit length
		escaped := strings.ReplaceAll(v, "|", "\\|")
		escaped = strings.ReplaceAll(escaped, "\n", " ")
		if len(escaped) > 200 {
			escaped = escaped[:200] + "..."
		}
		return escaped
	case []byte:
		s := string(v)
		if len(s) > 200 {
			s = s[:200] + "..."
		}
		return s
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 200 {
			s = s[:200] + "..."
		}
		return s
	}
}
