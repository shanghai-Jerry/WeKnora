package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// QueryDataSourceHandler handles HTTP requests for AI query data source management
type QueryDataSourceHandler struct {
	service interfaces.QueryDataSourceService
}

// NewQueryDataSourceHandler creates a new query data source handler
func NewQueryDataSourceHandler(service interfaces.QueryDataSourceService) *QueryDataSourceHandler {
	return &QueryDataSourceHandler{service: service}
}

// Create godoc
// @Summary Create a new query data source
// @Description Create a new data source for AI query (sql_query tool)
// @Tags QueryDataSource
// @Accept json
// @Produce json
// @Param request body types.QueryDataSource true "Query data source configuration"
// @Success 201 {object} types.QueryDataSource
// @Failure 400 {object} map[string]string
// @Router /api/v1/query-data-sources [post]
func (h *QueryDataSourceHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant ID from context (set by auth middleware)
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: tenant context missing"})
		return
	}

	var req types.QueryDataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate required fields
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	// Set tenant ID
	req.TenantID = tenantID

	// Create
	result, err := h.service.Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// List godoc
// @Summary List query data sources
// @Description List all query data sources for the current tenant
// @Tags QueryDataSource
// @Produce json
// @Success 200 {array} types.QueryDataSource
// @Failure 500 {object} map[string]string
// @Router /api/v1/query-data-sources [get]
func (h *QueryDataSourceHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant ID from context
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: tenant context missing"})
		return
	}

	// List
	result, err := h.service.ListByTenant(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Get godoc
// @Summary Get a query data source
// @Description Get a query data source by ID
// @Tags QueryDataSource
// @Produce json
// @Param id path string true "Query data source ID"
// @Success 200 {object} types.QueryDataSource
// @Failure 404 {object} map[string]string
// @Router /api/v1/query-data-sources/{id} [get]
func (h *QueryDataSourceHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	result, err := h.service.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Update godoc
// @Summary Update a query data source
// @Description Update a query data source by ID
// @Tags QueryDataSource
// @Accept json
// @Produce json
// @Param id path string true "Query data source ID"
// @Param request body types.QueryDataSource true "Query data source configuration"
// @Success 200 {object} types.QueryDataSource
// @Failure 400 {object} map[string]string
// @Router /api/v1/query-data-sources/{id} [put]
func (h *QueryDataSourceHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	var req types.QueryDataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Set ID
	req.ID = id

	// Update
	result, err := h.service.Update(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Delete godoc
// @Summary Delete a query data source
// @Description Delete a query data source by ID
// @Tags QueryDataSource
// @Param id path string true "Query data source ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /api/v1/query-data-sources/{id} [delete]
func (h *QueryDataSourceHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ValidateConnection godoc
// @Summary Validate connection to a data source
// @Description Test connection to a database data source
// @Tags QueryDataSource
// @Accept json
// @Produce json
// @Param request body types.QueryDataSource true "Query data source configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/query-data-sources/validate [post]
func (h *QueryDataSourceHandler) ValidateConnection(c *gin.Context) {
	ctx := c.Request.Context()

	var req types.QueryDataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.service.ValidateConnection(ctx, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "connection successful"})
}
