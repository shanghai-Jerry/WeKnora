package session

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// ServeSandboxFile serves files from the sandbox workspace for a given session.
// Route: GET /api/v1/sessions/:id/sandbox-files/*filepath
func (h *Handler) ServeSandboxFile(c *gin.Context) {
	ctx := c.Request.Context()

	sessionID := secutils.SanitizeForLog(c.Param("id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session id"})
		return
	}

	filePath := strings.TrimPrefix(c.Param("filepath"), "/")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file path"})
		return
	}

	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	sandboxBase := os.Getenv("WEKNORA_SANDBOX_WORKSPACE")
	if sandboxBase == "" {
		sandboxBase = "/data/sandbox"
	}

	sessionDir := filepath.Join(sandboxBase, sessionID)
	absSessionDir, err := filepath.Abs(sessionDir)
	if err != nil {
		logger.Errorf(ctx, "Failed to resolve sandbox session dir: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	targetPath := filepath.Join(absSessionDir, filePath)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		logger.Errorf(ctx, "Failed to resolve target file path: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Ensure the resolved path stays within the session directory
	if !strings.HasPrefix(absTarget, absSessionDir+string(filepath.Separator)) && absTarget != absSessionDir {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	info, err := os.Stat(absTarget)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(absTarget))
	contentType := "application/octet-stream"
	switch ext {
	case ".html", ".htm":
		contentType = "text/html; charset=utf-8"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".svg":
		contentType = "image/svg+xml"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".json":
		contentType = "application/json; charset=utf-8"
	case ".csv":
		contentType = "text/csv; charset=utf-8"
	case ".pdf":
		contentType = "application/pdf"
	}

	c.Header("Content-Type", contentType)
	c.File(absTarget)
}
