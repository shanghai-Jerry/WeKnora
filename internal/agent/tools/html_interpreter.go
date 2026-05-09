package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

var htmlInterpreterTool = BaseTool{
	name: ToolHtmlInterpreter,
	description: `Render HTML as an interactive web report.

## Modes
1. **Inline HTML** (default): Pass HTML directly in the "html" parameter
2. **File mode**: Read HTML from a file in the work directory using "file_path"
3. **Template mode**: Read a template file and replace {{KEY}} placeholders with data

## Usage
- Provide HTML content directly, or
- Provide a file_path to read HTML from the work directory, or
- Provide a file_path + data map for template placeholder replacement

## When to Use
- After code_interpreter generates data/charts, use this to render a visual report
- To display HTML content, charts, or interactive dashboards
- For generating formatted reports from data

## Notes
- HTML is rendered as a self-contained document
- Generated images from code_interpreter can be referenced in the HTML
- The report will be displayed in a side panel`,
	schema: utils.GenerateSchema[HtmlInterpreterInput](),
}

// HtmlInterpreterInput defines the input parameters for the html_interpreter tool
type HtmlInterpreterInput struct {
	HTML     string            `json:"html,omitempty" jsonschema:"Inline HTML content"`
	Title    string            `json:"title,omitempty" jsonschema:"Report title"`
	Data     map[string]string `json:"data,omitempty" jsonschema:"Template data for placeholder replacement (key-value pairs, replaced as {{KEY}} -> value)"`
	FilePath string            `json:"file_path,omitempty" jsonschema:"Path to HTML file to read from work directory"`
}

// HtmlInterpreterTool renders HTML content as interactive reports
type HtmlInterpreterTool struct {
	BaseTool
	sessionID string
	workDir   string
}

// NewHtmlInterpreterTool creates a new html_interpreter tool instance
func NewHtmlInterpreterTool(sessionID string, workDir string) *HtmlInterpreterTool {
	return &HtmlInterpreterTool{
		BaseTool: htmlInterpreterTool,
		sessionID: sessionID,
		workDir:   workDir,
	}
}

// Execute renders the provided HTML content
func (t *HtmlInterpreterTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][HtmlInterpreter] Execute started")

	var input HtmlInterpreterInput
	if err := json.Unmarshal(args, &input); err != nil {
		logger.Errorf(ctx, "[Tool][HtmlInterpreter] Failed to parse args: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse input args: %v", err),
		}, nil
	}

	var htmlContent string

	switch {
	case input.FilePath != "":
		htmlContent = t.resolveFromFile(ctx, input.FilePath, input.Data)
		if htmlContent == "" {
			return &types.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to read HTML file: %s", input.FilePath),
			}, nil
		}

	case input.HTML != "":
		htmlContent = input.HTML

	default:
		return &types.ToolResult{
			Success: false,
			Error:   "No HTML content provided. Please provide 'html' content or a 'file_path' to read from.",
		}, nil
	}

	htmlContent = strings.ReplaceAll(htmlContent, "\\n", "\n")
	htmlContent = strings.ReplaceAll(htmlContent, "\\t", "\t")

	if !strings.Contains(htmlContent, "<!DOCTYPE") && !strings.Contains(htmlContent, "<html") {
		htmlContent = "<!DOCTYPE html>\n<html lang=\"en\">\n<head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"></head>\n<body>\n" + htmlContent + "\n</body>\n</html>"
	}

	title := input.Title
	if title == "" {
		title = "HTML Report"
	}

	logger.Infof(ctx, "[Tool][HtmlInterpreter] Rendering HTML (%d bytes), title: %s", len(htmlContent), title)

	resultData := map[string]interface{}{
		"output_type": "html",
		"title":       title,
	}

	return &types.ToolResult{
		Success: true,
		Output:  htmlContent,
		Data:    resultData,
	}, nil
}

// resolveFromFile reads HTML content from a file in the work directory
func (t *HtmlInterpreterTool) resolveFromFile(ctx context.Context, filePath string, data map[string]string) string {
	fullPath := filepath.Join(t.workDir, filePath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.Warnf(ctx, "[Tool][HtmlInterpreter] File not found: %s", fullPath)
		return ""
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		logger.Errorf(ctx, "[Tool][HtmlInterpreter] Failed to read file %s: %v", fullPath, err)
		return ""
	}

	htmlContent := string(content)

	if len(data) > 0 {
		htmlContent = replacePlaceholders(htmlContent, data)
	}

	return htmlContent
}

// replacePlaceholders replaces {{KEY}} placeholders with values from data map
func replacePlaceholders(template string, data map[string]string) string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		key := match[2 : len(match)-2]
		if val, ok := data[key]; ok {
			return val
		}
		return match
	})
}
