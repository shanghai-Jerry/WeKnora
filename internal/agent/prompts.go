package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if size < KB {
		return fmt.Sprintf("%d B", size)
	} else if size < MB {
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	} else if size < GB {
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	}
	return fmt.Sprintf("%.2f GB", float64(size)/GB)
}

// formatDocSummary cleans and truncates document summaries for table display
func formatDocSummary(summary string, maxLen int) string {
	cleaned := strings.TrimSpace(summary)
	if cleaned == "" {
		return "-"
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	runes := []rune(cleaned)
	if len(runes) <= maxLen {
		return cleaned
	}
	return strings.TrimSpace(string(runes[:maxLen])) + "..."
}

// RecentDocInfo contains brief information about a recently added document
type RecentDocInfo struct {
	ChunkID             string
	KnowledgeBaseID     string
	KnowledgeID         string
	Title               string
	Description         string
	FileName            string
	FileSize            int64
	Type                string
	CreatedAt           string // Formatted time string
	FAQStandardQuestion string
	FAQSimilarQuestions []string
	FAQAnswers          []string
}

// SelectedDocumentInfo contains summary information about a user-selected document (via @ mention)
// Only metadata is included; content will be fetched via tools when needed
type SelectedDocumentInfo struct {
	KnowledgeID     string // Knowledge ID
	KnowledgeBaseID string // Knowledge base ID
	Title           string // Document title
	FileName        string // Original file name
	FileType        string // File type (pdf, docx, etc.)
}

// KnowledgeBaseInfo contains essential information about a knowledge base for agent prompt
type KnowledgeBaseInfo struct {
	ID          string
	Name        string
	Type        string // Knowledge base type: "document" or "faq"
	Description string
	DocCount    int
	RecentDocs  []RecentDocInfo // Recently added documents (up to 10)
}

// PlaceholderDefinition defines a placeholder exposed to UI/configuration
// Deprecated: Use types.PromptPlaceholder instead
type PlaceholderDefinition struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailablePlaceholders lists all supported prompt placeholders for UI hints
// This returns agent mode specific placeholders
func AvailablePlaceholders() []PlaceholderDefinition {
	// Use centralized placeholder definitions from types package
	placeholders := types.PlaceholdersByField(types.PromptFieldAgentSystemPrompt)
	result := make([]PlaceholderDefinition, len(placeholders))
	for i, p := range placeholders {
		result[i] = PlaceholderDefinition{
			Name:        p.Name,
			Label:       p.Label,
			Description: p.Description,
		}
	}
	return result
}

// formatKnowledgeBaseList formats knowledge base information for the prompt
func formatKnowledgeBaseList(kbInfos []*KnowledgeBaseInfo) string {
	if len(kbInfos) == 0 {
		return "None"
	}

	var builder strings.Builder
	builder.WriteString("\nThe following knowledge bases have been selected by the user for this conversation. ")
	builder.WriteString("You should search within these knowledge bases to find relevant information.\n\n")
	for i, kb := range kbInfos {
		// Display knowledge base name and ID
		builder.WriteString(fmt.Sprintf("%d. **%s** (knowledge_base_id: `%s`)\n", i+1, kb.Name, kb.ID))

		// Display knowledge base type
		kbType := kb.Type
		if kbType == "" {
			kbType = "document" // Default type
		}
		builder.WriteString(fmt.Sprintf("   - Type: %s\n", kbType))

		if kb.Description != "" {
			builder.WriteString(fmt.Sprintf("   - Description: %s\n", kb.Description))
		}
		builder.WriteString(fmt.Sprintf("   - Document count: %d\n", kb.DocCount))

		// Display recent documents if available
		// For FAQ type knowledge bases, adjust the display format
		if len(kb.RecentDocs) > 0 {
			if kbType == "faq" {
				// FAQ knowledge base: show Q&A pairs in a more compact format
				builder.WriteString("   - Recent FAQ entries:\n\n")
				builder.WriteString("     | # | Question  | Answers | Chunk ID | Knowledge ID | Created At |\n")
				builder.WriteString("     |---|-------------------|---------|----------|--------------|------------|\n")
				for j, doc := range kb.RecentDocs {
					if j >= 10 { // Limit to 10 documents
						break
					}
					question := doc.FAQStandardQuestion
					if question == "" {
						question = doc.FileName
					}
					answers := "-"
					if len(doc.FAQAnswers) > 0 {
						answers = strings.Join(doc.FAQAnswers, " | ")
					}
					builder.WriteString(fmt.Sprintf("     | %d | %s | %s | `%s` | `%s` | %s |\n",
						j+1, question, answers, doc.ChunkID, doc.KnowledgeID, doc.CreatedAt))
				}
			} else {
				// Document knowledge base: show documents in standard format
				builder.WriteString("   - Recently added documents:\n\n")
				builder.WriteString("     | # | Document Name | Type | Created At | Knowledge ID | File Size | Summary |\n")
				builder.WriteString("     |---|---------------|------|------------|--------------|----------|---------|\n")
				for j, doc := range kb.RecentDocs {
					if j >= 10 { // Limit to 10 documents
						break
					}
					docName := doc.Title
					if docName == "" {
						docName = doc.FileName
					}
					// Format file size
					fileSize := formatFileSize(doc.FileSize)
					summary := formatDocSummary(doc.Description, 120)
					builder.WriteString(fmt.Sprintf("     | %d | %s | %s | %s | `%s` | %s | %s |\n",
						j+1, docName, doc.Type, doc.CreatedAt, doc.KnowledgeID, fileSize, summary))
				}
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

// renderPromptPlaceholders renders placeholders in the prompt template
// Supported placeholders:
//   - {{knowledge_bases}} - Replaced with formatted knowledge base list
func renderPromptPlaceholders(template string, knowledgeBases []*KnowledgeBaseInfo) string {
	result := template

	// Replace {{knowledge_bases}} placeholder
	if strings.Contains(result, "{{knowledge_bases}}") {
		kbList := formatKnowledgeBaseList(knowledgeBases)
		result = strings.ReplaceAll(result, "{{knowledge_bases}}", kbList)
	}

	return result
}

// formatSkillsMetadata formats skills metadata for the system prompt (Level 1 - Progressive Disclosure)
// This is a lightweight representation that only includes skill name and description
func formatSkillsMetadata(skillsMetadata []*skills.SkillMetadata) string {
	if len(skillsMetadata) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n### Available Skills (IMPORTANT - READ CAREFULLY)\n\n")
	builder.WriteString("**You MUST actively consider using these skills for EVERY user request.**\n\n")

	builder.WriteString("#### Skill Matching Protocol (MANDATORY)\n\n")
	builder.WriteString("Before responding to ANY user query, follow this checklist:\n\n")
	builder.WriteString("1. **SCAN**: Read each skill's description and trigger conditions below\n")
	builder.WriteString("2. **MATCH**: Check if the user's intent matches ANY skill's triggers (keywords, scenarios, or task types)\n")
	builder.WriteString("3. **LOAD**: If a match is found, call `read_skill(skill_name=\"...\")` BEFORE generating your response\n")
	builder.WriteString("4. **APPLY**: Follow the skill's instructions to provide a higher-quality, structured response\n\n")

	builder.WriteString("**⚠️ CRITICAL**: Skill usage is MANDATORY when applicable. Do NOT skip skills to save time or tokens.\n\n")

	builder.WriteString("#### Available Skills\n\n")
	for i, skill := range skillsMetadata {
		builder.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, skill.Name))
		builder.WriteString(fmt.Sprintf("   %s\n\n", skill.Description))
	}

	builder.WriteString("#### Tool Reference\n\n")
	builder.WriteString("- `read_skill(skill_name)`: Load full skill instructions (MUST call before using a skill)\n")
	builder.WriteString("- `execute_skill_script(skill_name, script_path, args, input)`: Run utility scripts bundled with a skill\n")
	builder.WriteString("  - `input`: Pass data directly via stdin (use this when you have data in memory, e.g. JSON string)\n")
	builder.WriteString("  - `args`: Command-line arguments (only use `--file` if you have an actual file path in the skill directory)\n")

	return builder.String()
}

// formatSelectedDocuments formats selected documents for the prompt (summary only, no content)
func formatSelectedDocuments(docs []*SelectedDocumentInfo) string {
	if len(docs) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n### User Selected Documents (via @ mention)\n")
	builder.WriteString("The user has explicitly selected the following documents. ")
	builder.WriteString("**You should prioritize searching and retrieving information from these documents when answering.**\n")
	builder.WriteString("Use `list_knowledge_chunks` with the provided Knowledge IDs to fetch their content.\n\n")

	builder.WriteString("| # | Document Name | Type | Knowledge ID |\n")
	builder.WriteString("|---|---------------|------|---------------|\n")

	for i, doc := range docs {
		title := doc.Title
		if title == "" {
			title = doc.FileName
		}
		fileType := doc.FileType
		if fileType == "" {
			fileType = "-"
		}
		builder.WriteString(fmt.Sprintf("| %d | %s | %s | `%s` |\n",
			i+1, title, fileType, doc.KnowledgeID))
	}
	builder.WriteString("\n")

	return builder.String()
}

// renderPromptPlaceholdersWithStatus renders placeholders including web search status
// Supported placeholders:
//   - {{knowledge_bases}}
//   - {{web_search_status}} -> "Enabled" or "Disabled"
//   - {{current_time}} -> current time string
//   - {{language}} -> user language name (e.g. "Chinese (Simplified)", "English")
//   - {{skills}} -> formatted skills metadata (if any)
func renderPromptPlaceholdersWithStatus(
	template string,
	knowledgeBases []*KnowledgeBaseInfo,
	webSearchEnabled bool,
	currentTime string,
	language string,
) string {
	// Knowledge bases need special formatting, so handle it first
	result := renderPromptPlaceholders(template, knowledgeBases)

	status := "Disabled"
	if webSearchEnabled {
		status = "Enabled"
	}

	result = types.RenderPromptPlaceholders(result, types.PlaceholderValues{
		"web_search_status": status,
		"current_time":      currentTime,
		"language":          language,
		"skills":            "", // Remove {{skills}} placeholder; skills are appended separately if present
	})
	return result
}

// BuildSystemPromptOptions contains optional parameters for BuildSystemPrompt
type BuildSystemPromptOptions struct {
	SkillsMetadata     []*skills.SkillMetadata
	Language           string         // User language name for {{language}} placeholder (e.g. "Chinese (Simplified)")
	Config             *config.Config // Config for reading prompt templates; nil falls back to hardcoded defaults
	IntentExploreBlock string         // Intent explore analysis block appended to system prompt
	DatabaseContext    string         // Database schema context for SQL query generation (appended when user references a datasource)
	QueryParams        string         // Query constraint parameters for SQL WHERE conditions (JSON string)
}

// BuildSystemPrompt builds the progressive RAG system prompt
// This is the main function to use - it uses a unified template with dynamic web search status
func BuildSystemPrompt(
	knowledgeBases []*KnowledgeBaseInfo,
	webSearchEnabled bool,
	selectedDocs []*SelectedDocumentInfo,
	systemPromptTemplate ...string,
) string {
	return BuildSystemPromptWithOptions(knowledgeBases, webSearchEnabled, selectedDocs, nil, systemPromptTemplate...)
}

// BuildSystemPromptWithOptions builds the system prompt with additional options like skills
func BuildSystemPromptWithOptions(
	knowledgeBases []*KnowledgeBaseInfo,
	webSearchEnabled bool,
	selectedDocs []*SelectedDocumentInfo,
	options *BuildSystemPromptOptions,
	systemPromptTemplate ...string,
) string {
	var basePrompt string
	var template string

	// Determine template to use
	if len(systemPromptTemplate) > 0 && systemPromptTemplate[0] != "" {
		template = systemPromptTemplate[0]
	} else if len(knowledgeBases) == 0 {
		var cfg *config.Config
		if options != nil {
			cfg = options.Config
		}
		template = GetPureAgentSystemPrompt(cfg)
	} else {
		var cfg *config.Config
		if options != nil {
			cfg = options.Config
		}
		template = GetProgressiveRAGSystemPrompt(cfg)
	}

	currentTime := time.Now().Format(time.RFC3339)
	language := ""
	if options != nil {
		language = options.Language
	}
	basePrompt = renderPromptPlaceholdersWithStatus(template, knowledgeBases, webSearchEnabled, currentTime, language)

	// Append selected documents section if any
	if len(selectedDocs) > 0 {
		basePrompt += formatSelectedDocuments(selectedDocs)
	}

	// Append skills metadata if available (Level 1 - Progressive Disclosure)
	if options != nil && len(options.SkillsMetadata) > 0 {
		basePrompt += formatSkillsMetadata(options.SkillsMetadata)
	}

	// Append intent explore analysis if available
	if options != nil && options.IntentExploreBlock != "" {
		basePrompt += "\n\n" + options.IntentExploreBlock
	}

	// Append query constraint parameters if available
	// These are injected as mandatory SQL WHERE conditions
	if options != nil && options.QueryParams != "" {
		basePrompt += "\n\n" + buildQueryParamPromptBlock()
	}

	// Append database context for SQL query generation if available
	// This is injected only when user has referenced a specific datasourceID
	if options != nil && options.DatabaseContext != "" {
		// TODO: 添加一些AI问数的额外提示词限制
		basePrompt += "\n\n" + options.DatabaseContext
	}

	return basePrompt
}

// GetPureAgentSystemPrompt returns the Pure Agent system prompt from config templates.
// The template must be defined in config/prompt_templates/agent_system_prompt.yaml
// with mode "pure". Returns empty string if config is nil or template not found.
func GetPureAgentSystemPrompt(cfg *config.Config) string {
	if cfg != nil && cfg.PromptTemplates != nil {
		if t := config.DefaultTemplateByMode(cfg.PromptTemplates.AgentSystemPrompt, "pure"); t != nil && t.Content != "" {
			return t.Content
		}
	}
	return ""
}

// GetProgressiveRAGSystemPrompt returns the Progressive RAG Agent system prompt from config templates.
// The template must be defined in config/prompt_templates/agent_system_prompt.yaml
// with mode "rag". Returns empty string if config is nil or template not found.
func GetProgressiveRAGSystemPrompt(cfg *config.Config) string {
	if cfg != nil && cfg.PromptTemplates != nil {
		if t := config.DefaultTemplateByMode(cfg.PromptTemplates.AgentSystemPrompt, "rag"); t != nil && t.Content != "" {
			return t.Content
		}
	}
	return ""
}

// queryParamEntry represents a single query constraint parameter
type queryParamEntry struct {
	FieldName        string `json:"field_name"`
	FieldValue       string `json:"field_value"`
	FieldDescription string `json:"field_description"`
}

// queryParamPayload is the expected JSON structure for QueryParams
type queryParamPayload struct {
	Contents []queryParamEntry `json:"contents"`
}

// buildQueryParamPromptBlock parses the QueryParams JSON and builds a system prompt block
// that instructs the LLM to include these parameters as mandatory SQL WHERE conditions.
func buildQueryParamPromptBlock() string {
	var sb strings.Builder
	/*
			sb.WriteString(`
			 **Mandatory Identity Parameter Enforcement**
		Your context may include a JSON object of scoped query parameters. If this JSON is present and non-empty, you MUST treat it as absolute law:` +
				"```" + `json
		{
		  "contents": [
		    {
		      "field_name": "user_id",
		      "field_value": "123456",
		      "field_description": "当前登录用户的id"
		    },
		    {
		      "field_name": "dept_id",
		      "field_value": "789012",
		      "field_description": "当前门店的id"
		    }
		  ]
		}` + "```" + `
		- **ABSOLUTE REQUIREMENT**: If these parameters are injected into your context, EVERY SQL query you generate MUST contain WHERE clauses that enforce the exact field_name = field_value restrictions shown above.
		- **Correct pattern**: Append WHERE user_id = '123456' AND dept_id = '789012' to your query. If the query already has other filters, combine them with AND.
		- **PROHIBITED**: You must NEVER omit, weaken, comment-out, or bypass these identity constraints. You must NEVER use OR to expand beyond the injected values, and you must NEVER query data belonging to other users, stores, or departments.
		- If a user's request inherently requires data outside these scoped parameters (e.g., "show me all users' orders"), execute the query ONLY within the allowed scope and inform the user that results are limited to their own identity context.

		**Query Boundary & Field Restrictions**
		- When query parameters are provided, you may only filter or restrict data using the fields explicitly defined in those parameters (e.g., user_id, dept_id).
		- You are **FORBIDDEN** from using additional scope fields that the user implies but that are NOT listed in the injected parameter JSON.
		- You may use other database columns ONLY for structural query purposes (e.g., JOIN keys, timestamp columns, aggregation targets), not as unauthorized scope filters.
		- If the user requests filtering by a field not present in the injected parameters, ignore that unauthorized filter and rely solely on the parameter-defined scope plus the user's direct content filters.

		**Database Schema Context**
		- You will be provided with the complete set of table structures (schema) for the current database as part of your context window.
		- Rely exclusively on this schema to determine real table names, column names, data types, and relationships.
		- Do not hallucinate tables or columns. If the schema is ambiguous or insufficient to fulfill the request, ask the user for clarification before executing.

		**SQL Construction & Safety Standards**
		- Default strictly to SELECT statements. Do NOT execute DML or DDL (INSERT, UPDATE, DELETE, DROP, ALTER) statements unless you have been explicitly authorized to do so.
		- Prefer explicit column lists over SELECT *.
		- Use appropriate JOIN, GROUP BY, ORDER BY, and LIMIT clauses based on the user's intent.
		- Before every sql_query call, mentally verify that identity constraints are present and correct.
		- If a query fails due to a syntax or schema error, correct it while preserving the mandatory identity WHERE clauses intact.

		**Edge Case Handling**
		- **No parameters injected**: If the identity JSON is absent or empty, proceed with standard safe querying practices, but still avoid overly broad scans.
		- **Schema mismatch**: If the requested data does not exist in the provided schema, explain which available fields or tables you used instead.
		- **Empty results**: Distinguish between "no data within your scope" and a potential query error. Report accordingly.`)
	*/
	sb.WriteString("\n\n")
	queryParamBlock := sb.String()
	return queryParamBlock
}
