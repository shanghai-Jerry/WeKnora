package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/datasource"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/utils"
	_ "github.com/go-sql-driver/mysql"
)

const (
	// ConnectorType is the identifier for the MySQL connector
	ConnectorType = "mysql"

	// DefaultPort is the default MySQL port
	DefaultPort = 3306

	// DefaultCharset is the default character set
	DefaultCharset = "utf8mb4"

	// DefaultMaxRows is the default maximum number of rows to return
	DefaultMaxRows = 50

	// ConnectionTimeout is the timeout for establishing a connection
	ConnectionTimeout = 10 * time.Second

	// QueryTimeout is the timeout for executing queries
	QueryTimeout = 30 * time.Second
)

// Config represents the MySQL connection configuration
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Charset  string `json:"charset"`
}

// Connector implements the datasource.DBConnector interface for MySQL
type Connector struct{}

// NewConnector creates a new MySQL connector
func NewConnector() *Connector {
	return &Connector{}
}

// Type returns the connector type identifier
func (c *Connector) Type() string {
	return ConnectorType
}

// ValidateConnection tests the MySQL connection with the provided config
func (c *Connector) ValidateConnection(ctx context.Context, config map[string]interface{}) error {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	logger.Infof(ctx, "[MySQL] Connection validated successfully to %s:%d/%s",
		mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database)
	return nil
}

// ExecuteQuery executes a read-only SQL query and returns results
func (c *Connector) ExecuteQuery(ctx context.Context, config map[string]interface{}, query string, maxRows int) (*datasource.QueryResult, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql config: %w", err)
	}

	// Validate query - only allow SELECT statements
	if err := validateReadOnlyQuery(query); err != nil {
		return nil, err
	}

	// Set default max rows if not specified
	if maxRows <= 0 {
		maxRows = DefaultMaxRows
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	// Create a context with timeout for the query
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Get total count first (wrap query in a count subquery)
	totalCount, err := getTotalCount(queryCtx, db, query)
	if err != nil {
		logger.Warnf(ctx, "[MySQL] Failed to get total count: %v", err)
		// Continue without total count
	}

	// Add LIMIT to query if not already present
	limitedQuery := addLimitToQuery(query, maxRows)

	logger.Infof(ctx, "[MySQL] Executing query: %s", limitedQuery)

	// Execute the query
	rows, err := db.QueryContext(queryCtx, limitedQuery)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process results
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		// Scan the row
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for this row
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			// Convert []byte to string for better readability
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	logger.Infof(ctx, "[MySQL] Query returned %d rows (total: %d)", len(results), totalCount)

	return &datasource.QueryResult{
		Columns:    columns,
		Rows:       results,
		RowCount:   len(results),
		TotalCount: totalCount,
	}, nil
}

// GetTableSchema returns schema information for all accessible tables
func (c *Connector) GetTableSchema(ctx context.Context, config map[string]interface{}) ([]datasource.TableInfo, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	// Query to get all tables
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	rows, err := db.QueryContext(queryCtx,
		"SELECT TABLE_NAME, TABLE_COMMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'",
		mysqlConfig.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []datasource.TableInfo
	for rows.Next() {
		var tableName, tableComment string
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		// Get columns for this table
		tableInfo, err := c.getTableColumns(queryCtx, db, mysqlConfig.Database, tableName)
		if err != nil {
			logger.Warnf(ctx, "[MySQL] Failed to get columns for table %s: %v", tableName, err)
			continue
		}
		tableInfo.Description = tableComment
		tables = append(tables, *tableInfo)
	}

	return tables, nil
}

// GetTableSchemaForTable returns schema information for a specific table
func (c *Connector) GetTableSchemaForTable(ctx context.Context, config map[string]interface{}, tableName string) (*datasource.TableInfo, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	return c.getTableColumns(queryCtx, db, mysqlConfig.Database, tableName)
}

// getTableColumns retrieves column information for a specific table
func (c *Connector) getTableColumns(ctx context.Context, db *sql.DB, database, tableName string) (*datasource.TableInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_COMMENT, COLUMN_KEY
		 FROM INFORMATION_SCHEMA.COLUMNS
		 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		 ORDER BY ORDINAL_POSITION`,
		database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	tableInfo := &datasource.TableInfo{
		Name: tableName,
	}

	for rows.Next() {
		var columnName, columnType, isNullable, columnKey string
		var columnComment sql.NullString
		if err := rows.Scan(&columnName, &columnType, &isNullable, &columnComment, &columnKey); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columnInfo := datasource.ColumnInfo{
			Name:         columnName,
			Type:         columnType,
			Nullable:     isNullable == "YES",
			IsPrimaryKey: columnKey == "PRI",
		}
		if columnComment.Valid {
			columnInfo.Description = columnComment.String
		}
		tableInfo.Columns = append(tableInfo.Columns, columnInfo)
	}

	return tableInfo, nil
}

// parseConfig parses the config map into a Config struct
func parseConfig(config map[string]interface{}) (*Config, error) {
	host, _ := config["host"].(string)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	port, _ := config["port"].(float64)
	if port == 0 {
		port = DefaultPort
	}

	username, _ := config["username"].(string)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	password, _ := config["password"].(string)
	database, _ := config["database"].(string)
	if database == "" {
		return nil, fmt.Errorf("database is required")
	}

	charset, _ := config["charset"].(string)
	if charset == "" {
		charset = DefaultCharset
	}

	return &Config{
		Host:     host,
		Port:     int(port),
		Username: username,
		Password: password,
		Database: database,
		Charset:  charset,
	}, nil
}

// openConnection opens a new MySQL connection
func openConnection(ctx context.Context, config *Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local&timeout=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset,
		ConnectionTimeout,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// validateReadOnlyQuery validates that the query is read-only
func validateReadOnlyQuery(query string) error {
	// Strip comments before checking
	cleaned := utils.StripSQLComments(query)
	// Trim whitespace and convert to uppercase for checking
	trimmed := strings.TrimSpace(strings.ToUpper(cleaned))

	// Check for forbidden keywords that indicate write operations
	forbiddenKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
		"TRUNCATE", "REPLACE", "RENAME", "GRANT", "REVOKE",
		"CALL", "EXEC", "EXECUTE", "SET", "LOAD", "IMPORT",
	}

	for _, keyword := range forbiddenKeywords {
		// Check if the keyword appears at the start of the query (after trimming)
		if strings.HasPrefix(trimmed, keyword) {
			return fmt.Errorf("forbidden operation: %s statements are not allowed. Only SELECT queries are permitted", keyword)
		}
		// Also check for keyword preceded by semicolon (multiple statements)
		if strings.Contains(trimmed, ";"+keyword) {
			return fmt.Errorf("forbidden operation: multiple statements with %s are not allowed", keyword)
		}
	}

	// Only allow read-only statements
	allowedPrefixes := []string{"SELECT", "WITH", "DESCRIBE", "SHOW"}
	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("invalid query: only SELECT, DESCRIBE, and SHOW statements are allowed")
	}

	return nil
}

// getTotalCount gets the total count of rows for a query
func getTotalCount(ctx context.Context, db *sql.DB, query string) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS _count_subquery", query)

	var count int
	err := db.QueryRowContext(ctx, countQuery).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// addLimitToQuery adds a LIMIT clause to the query if not already present
func addLimitToQuery(query string, maxRows int) string {
	upperQuery := strings.ToUpper(strings.TrimSpace(query))

	// Check if LIMIT is already present
	if strings.Contains(upperQuery, "LIMIT") {
		return query
	}

	// Remove trailing semicolon if present
	query = strings.TrimRight(strings.TrimSpace(query), ";")

	return fmt.Sprintf("%s LIMIT %d", query, maxRows)
}

// GetCreateTableSQL returns the full CREATE TABLE statement for a specific table
// This executes SHOW CREATE TABLE and returns the complete DDL including column definitions
// and table comment
func (c *Connector) GetCreateTableSQL(ctx context.Context, config map[string]interface{}, tableName string) (string, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return "", fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Execute SHOW CREATE TABLE to get the complete DDL
	var tableNameRet, createTableSQL string
	err = db.QueryRowContext(queryCtx, "SHOW CREATE TABLE `"+tableName+"`").Scan(&tableNameRet, &createTableSQL)
	if err != nil {
		return "", fmt.Errorf("failed to get CREATE TABLE for %s: %w", tableName, err)
	}

	return createTableSQL, nil
}

// GetSampleData returns sample rows from a table (used for context injection)
// Returns the column names and up to maxRows rows of data
func (c *Connector) GetSampleData(ctx context.Context, config map[string]interface{}, tableName string, maxRows int) ([]string, []map[string]interface{}, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Set default max rows
	if maxRows <= 0 {
		maxRows = 3
	}

	// Execute SELECT * FROM table LIMIT maxRows
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d", tableName, maxRows)
	rows, err := db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query sample data from %s: %w", tableName, err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process rows
	var results []map[string]interface{}
	for rows.Next() {
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return columns, results, nil
}

// GetDatabaseContext returns a formatted database context string for LLM prompt injection.
// It fetches schema, DDL, and sample data using a single database connection for efficiency.
func (c *Connector) GetDatabaseContext(ctx context.Context, config map[string]interface{}, maxSampleRows int) (string, error) {
	mysqlConfig, err := parseConfig(config)
	if err != nil {
		return "", fmt.Errorf("invalid mysql config: %w", err)
	}

	db, err := openConnection(ctx, mysqlConfig)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Fetch all tables
	rows, err := db.QueryContext(queryCtx,
		"SELECT TABLE_NAME, TABLE_COMMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'",
		mysqlConfig.Database)
	if err != nil {
		return "", fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []datasource.TableInfo
	var tableNames []string
	for rows.Next() {
		var tableName, tableComment string
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			return "", fmt.Errorf("failed to scan table: %w", err)
		}
		tableNames = append(tableNames, tableName)

		// Get columns for this table
		tableInfo, err := c.getTableColumns(queryCtx, db, mysqlConfig.Database, tableName)
		if err != nil {
			logger.Warnf(ctx, "[MySQL] Failed to get columns for table %s: %v", tableName, err)
			continue
		}
		tableInfo.Description = tableComment
		tables = append(tables, *tableInfo)
	}

	if len(tables) == 0 {
		return "", nil
	}

	// Build the database context block matching AI问数.md format
	var sb strings.Builder
	sb.WriteString("## 数据库信息\n")
	sb.WriteString(fmt.Sprintf("- 数据库名: %s\n", mysqlConfig.Database))
	sb.WriteString(fmt.Sprintf("- 可用表: %s\n", strings.Join(tableNames, ", ")))
	sb.WriteString("- 表结构:\n\n")

	for _, table := range tables {
		// Get the full CREATE TABLE statement
		var tableNameRet, createTableSQL string
		err = db.QueryRowContext(queryCtx, "SHOW CREATE TABLE `"+table.Name+"`").Scan(&tableNameRet, &createTableSQL)
		if err != nil {
			logger.Warnf(ctx, "[MySQL] Failed to get CREATE TABLE for %s: %v, falling back to schema info", table.Name, err)
			createTableSQL = datasource.FormatFallbackCreateTable(table)
		}
		sb.WriteString(createTableSQL)
		sb.WriteString("\n\n")

		// Get sample data
		if maxSampleRows <= 0 {
			maxSampleRows = 3
		}
		query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d", table.Name, maxSampleRows)
		sampleRows, err := db.QueryContext(queryCtx, query)
		if err != nil {
			logger.Warnf(ctx, "[MySQL] Failed to get sample data for %s: %v, skipping", table.Name, err)
			continue
		}

		columns, err := sampleRows.Columns()
		if err != nil {
			sampleRows.Close()
			continue
		}

		var results []map[string]interface{}
		for sampleRows.Next() {
			columnValues := make([]interface{}, len(columns))
			columnPointers := make([]interface{}, len(columns))
			for i := range columnValues {
				columnPointers[i] = &columnValues[i]
			}
			if err := sampleRows.Scan(columnPointers...); err != nil {
				continue
			}
			rowMap := make(map[string]interface{})
			for i, colName := range columns {
				val := columnValues[i]
				if b, ok := val.([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = val
				}
			}
			results = append(results, rowMap)
		}
		sampleRows.Close()

		if len(results) > 0 {
			sb.WriteString(fmt.Sprintf("/*\n%d rows from %s table:\n", len(results), table.Name))
			sb.WriteString(strings.Join(columns, "\t"))
			sb.WriteString("\n")
			for _, row := range results {
				values := make([]string, len(columns))
				for i, col := range columns {
					val := row[col]
					if val == nil {
						values[i] = "None"
					} else {
						values[i] = fmt.Sprintf("%v", val)
					}
				}
				sb.WriteString(strings.Join(values, "\t"))
				sb.WriteString("\n")
			}
			sb.WriteString("*/\n\n")
		}
	}

	sb.WriteString("- 使用 'sql_query' 工具执行 SQL 查询\n")
	sb.WriteString("- **只允许 SELECT 查询，禁止 INSERT/UPDATE/DELETE/DROP/ALTER/TRUNCATE**\n")

	return sb.String(), nil
}

func init() {
	// Register the MySQL connector with the global registry
	if err := datasource.GlobalDBConnectorRegistry.Register(NewConnector()); err != nil {
		logger.Errorf(context.Background(), "[MySQL] Failed to register connector: %v", err)
	}
}
