package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const (
	maxStdoutLength   = 2000
	executionTimeout  = 60 * time.Second
	pythonPreambleFmt = `import json
import os
import pandas as pd
import numpy as np
PLOT_DIR = r"%s"
os.makedirs(PLOT_DIR, exist_ok=True)
`
	jsPreambleFmt = `const fs = require('fs');
const path = require('path');
const PLOT_DIR = r"%s";
if (!fs.existsSync(PLOT_DIR)) fs.mkdirSync(PLOT_DIR, { recursive: true });
`
)

var codeInterpreterTool = BaseTool{
	name: ToolCodeInterpreter,
	description: `Execute Python or JavaScript code in a sandboxed environment for data analysis, computation, and chart generation.

## Usage
- Provide the code to execute in the "code" parameter
- Specify the language: "python" (default) or "javascript"
- Use PLOT_DIR variable to save charts/images (already set to the working directory)
- Use FILE_PATH variable if a file was provided (available in the working directory)

## When to Use
- Data analysis and computation tasks
- Generating charts, plots, or visualizations
- Running Python/JavaScript code for calculations
- Processing data files

## Available Variables
- PLOT_DIR: Working directory for saving output files (charts, images)
- FILE_PATH: Path to the user-upvided file (if any)

## Output
- Code execution output (stdout)
- Generated images (auto-detected from PLOT_DIR)
- Exit code indicating success (0) or failure (non-zero)`,
	schema: utils.GenerateSchema[CodeInterpreterInput](),
}

// CodeInterpreterInput defines the input parameters for the code_interpreter tool
type CodeInterpreterInput struct {
	Code     string `json:"code" jsonschema:"Code to execute (Python or JavaScript)"`
	Language string `json:"language,omitempty" jsonschema:"Language: python (default) or javascript"`
	FilePath string `json:"file_path,omitempty" jsonschema:"Optional file path to make available as FILE_PATH variable"`
}

// CodeInterpreterTool executes code in a sandboxed environment
type CodeInterpreterTool struct {
	BaseTool
	sandboxMgr sandbox.Manager
	sessionID  string
}

// NewCodeInterpreterTool creates a new code_interpreter tool instance
func NewCodeInterpreterTool(sandboxMgr sandbox.Manager, sessionID string) *CodeInterpreterTool {
	return &CodeInterpreterTool{
		BaseTool:   codeInterpreterTool,
		sandboxMgr: sandboxMgr,
		sessionID:  sessionID,
	}
}

// Execute runs the provided code in the sandbox
func (t *CodeInterpreterTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][CodeInterpreter] Execute started")
	var input CodeInterpreterInput
	if err := json.Unmarshal(args, &input); err != nil {
		logger.Errorf(ctx, "[Tool][CodeInterpreter] Failed to parse args: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse input args: %v", err),
		}, nil
	}

	if strings.TrimSpace(input.Code) == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "No code provided. Please provide code to execute.",
		}, nil
	}

	lang := strings.ToLower(strings.TrimSpace(input.Language))
	if lang == "" {
		lang = "python"
	}

	var scriptExt, interpreter string
	switch lang {
	case "python", "py":
		lang = "python"
		scriptExt = ".py"
		interpreter = "python3"
	case "javascript", "js", "node":
		lang = "javascript"
		scriptExt = ".js"
		interpreter = "node"
	default:
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Unsupported language: %s. Supported: python, javascript", input.Language),
		}, nil
	}

	// Use /data/sandbox as base when available (shared Docker volume for sandbox execution),
	// otherwise fall back to relative path for local development.
	sandboxBase := os.Getenv("WEKNORA_SANDBOX_WORKSPACE")
	if sandboxBase == "" {
		sandboxBase = "/data/sandbox"
	}
	workDir := filepath.Join(sandboxBase, t.sessionID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		// Fallback to relative path if shared volume is not available
		workDir = filepath.Join("data", "tmp", t.sessionID)
		if err := os.MkdirAll(workDir, 0755); err != nil {
			logger.Errorf(ctx, "[Tool][CodeInterpreter] Failed to create work dir: %v", err)
			return &types.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to create working directory: %v", err),
			}, nil
		}
	}

	// Convert to absolute path (sandbox requires absolute paths)
	absWorkDir, absErr := filepath.Abs(workDir)
	if absErr != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to resolve absolute path: %v", absErr),
		}, nil
	}
	workDir = absWorkDir
	logger.Infof(ctx, "[Tool][CodeInterpreter] workDir: %s", workDir)

	existingImages := scanImages(workDir)

	var preamble string
	switch lang {
	case "python":
		preamble = fmt.Sprintf(pythonPreambleFmt, workDir)
	case "javascript":
		preamble = fmt.Sprintf(jsPreambleFmt, workDir)
	}

	if input.FilePath != "" {
		switch lang {
		case "python":
			preamble += fmt.Sprintf("FILE_PATH = r\"%s\"\n", input.FilePath)
		case "javascript":
			preamble += fmt.Sprintf("const FILE_PATH = r\"%s\";\n", input.FilePath)
		}
	}

	fullCode := preamble + input.Code

	scriptPath := filepath.Join(workDir, "_run"+scriptExt)
	if err := os.WriteFile(scriptPath, []byte(fullCode), 0644); err != nil {
		logger.Errorf(ctx, "[Tool][CodeInterpreter] Failed to write script: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to write script file: %v", err),
		}, nil
	}

	logger.Infof(ctx, "[Tool][CodeInterpreter] Executing %s code (%d bytes)", lang, len(fullCode))

	// Determine if we need a shared Docker volume (for Docker-in-Docker scenario)
	sharedVolume := ""
	if sandboxBase := os.Getenv("WEKNORA_SANDBOX_WORKSPACE"); sandboxBase != "" && sandboxBase != "data/tmp" {
		sharedVolume = "sandbox-workspace"
	} else if sandboxBase == "" {
		// Default shared volume name when using default /data/sandbox path
		if workDir != "" && len(workDir) > 12 && workDir[:12] == "/data/sandbox" {
			sharedVolume = "sandbox-workspace"
		}
	}

	result, err := t.sandboxMgr.Execute(ctx, &sandbox.ExecuteConfig{
		Script:         scriptPath,
		WorkDir:        workDir,
		Timeout:        executionTimeout,
		SkipValidation: true,
		SharedVolume:   sharedVolume,
		AllowedCmds:    []string{interpreter, "python", "python3", "node", "bash", "sh", "cat", "ls", "echo", "mkdir"},
	})
	if err != nil {
		logger.Errorf(ctx, "[Tool][CodeInterpreter] Execution failed: %v", err)
		os.Remove(scriptPath)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Execution failed: %v", err),
		}, nil
	}

	os.Remove(scriptPath)

	var stdout, stderr string
	if result.Stdout != "" {
		stdout = result.Stdout
	}
	if result.Stderr != "" {
		stderr = result.Stderr
	}

	if len(stdout) > maxStdoutLength {
		stdout = stdout[:maxStdoutLength] + fmt.Sprintf("\n\n... (output truncated, %d of %d characters shown)", maxStdoutLength, len(result.Stdout))
	}

	newImages := findNewImages(workDir, existingImages)

	var outputBuilder strings.Builder
	outputBuilder.WriteString("```" + lang + "\n")
	outputBuilder.WriteString(input.Code)
	outputBuilder.WriteString("\n```\n")

	if stdout != "" {
		outputBuilder.WriteString("\n## Output\n\n")
		outputBuilder.WriteString("```\n")
		outputBuilder.WriteString(stdout)
		if !strings.HasSuffix(stdout, "\n") {
			outputBuilder.WriteString("\n")
		}
		outputBuilder.WriteString("```\n")
	}

	if stderr != "" {
		outputBuilder.WriteString("\n## Stderr\n\n")
		outputBuilder.WriteString("```\n")
		outputBuilder.WriteString(stderr)
		if !strings.HasSuffix(stderr, "\n") {
			outputBuilder.WriteString("\n")
		}
		outputBuilder.WriteString("```\n")
	}

	if result.Killed {
		outputBuilder.WriteString("\n**Warning**: Execution was terminated (timeout or killed)\n")
	}

	if result.Error != "" {
		outputBuilder.WriteString("\n## Error\n\n")
		outputBuilder.WriteString(result.Error)
		outputBuilder.WriteString("\n")
	}

	if len(newImages) > 0 {
		outputBuilder.WriteString("\n## Generated Images\n\n")
		for _, img := range newImages {
			outputBuilder.WriteString(fmt.Sprintf("- %s\n", img))
		}
	}

	success := result.IsSuccess()

	resultData := map[string]interface{}{
		"stdout":      stdout,
		"stderr":      stderr,
		"exit_code":   result.ExitCode,
		"language":    lang,
		"duration_ms": result.Duration.Milliseconds(),
		"killed":      result.Killed,
		"images":      newImages,
		"work_dir":    workDir,
	}
	output := outputBuilder.String()
	logger.Infof(ctx, "[Tool][CodeInterpreter] Completed with exit code: %d, images: %d", result.ExitCode, len(newImages))

	if result.ExitCode != 0 {
		logger.Errorf(ctx, "[Tool][CodeInterpreter] Code exited with non-zero exit code: %d, error: %s", result.ExitCode, output)
	}

	return &types.ToolResult{
		Success: success,
		Output:  output,
		Data:    resultData,
		Error: func() string {
			if !success {
				if result.Error != "" {
					return result.Error
				}
				return fmt.Sprintf("Code exited with code %d", result.ExitCode)
			}
			return ""
		}(),
	}, nil
}

// scanImages returns a set of image filenames in the directory
func scanImages(dir string) map[string]bool {
	existing := make(map[string]bool)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return existing
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isImageFile(name) {
			existing[name] = true
		}
	}
	return existing
}

// findNewImages returns image filenames that appeared after execution.
// Relative filenames are returned so HTML reports in the same directory can reference them directly.
func findNewImages(dir string, before map[string]bool) []string {
	var newImages []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return newImages
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isImageFile(name) && !before[name] {
			newImages = append(newImages, name)
		}
	}
	return newImages
}

// isImageFile checks if a filename has an image extension
func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return true
	}
	return false
}
