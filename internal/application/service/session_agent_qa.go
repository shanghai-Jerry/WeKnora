package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	chatpipeline "github.com/Tencent/WeKnora/internal/application/service/chat_pipeline"
	llmcontext "github.com/Tencent/WeKnora/internal/application/service/llmcontext"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// AgentQA performs agent-based question answering with conversation history and streaming support
// customAgent is optional - if provided, uses custom agent configuration instead of tenant defaults
// summaryModelID is optional - if provided, overrides the model from customAgent config
func (s *sessionService) AgentQA(
	ctx context.Context,
	req *types.QARequest,
	eventBus *event.EventBus,
) error {
	sessionID := req.Session.ID
	sessionJSON, err := json.Marshal(req.Session)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal session, session ID: %s, error: %v", sessionID, err)
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// customAgent is required for AgentQA (handler has already done permission check for shared agent)
	if req.CustomAgent == nil {
		logger.Warnf(ctx, "Custom agent not provided for session: %s", sessionID)
		return errors.New("custom agent configuration is required for agent QA")
	}

	// Resolve retrieval tenant using shared helper
	agentTenantID := s.resolveRetrievalTenantID(ctx, req)
	logger.Infof(ctx, "Start agent-based question answering, session ID: %s, agent tenant ID: %d, query: %s, session: %s",
		sessionID, agentTenantID, req.Query, string(sessionJSON))

	var tenantInfo *types.Tenant
	if v := ctx.Value(types.TenantInfoContextKey); v != nil {
		tenantInfo, _ = v.(*types.Tenant)
	}
	// When agent belongs to another tenant (shared agent), use agent's tenant for KB/model scope; load tenantInfo if needed
	if tenantInfo == nil || tenantInfo.ID != agentTenantID {
		if s.tenantService != nil {
			if agentTenant, err := s.tenantService.GetTenantByID(ctx, agentTenantID); err == nil && agentTenant != nil {
				tenantInfo = agentTenant
				logger.Infof(ctx, "Using agent tenant info for retrieval scope, tenant ID: %d", agentTenantID)
			}
		}
	}
	if tenantInfo == nil {
		logger.Warnf(ctx, "Tenant info not available for agent tenant %d, proceeding with defaults", agentTenantID)
		tenantInfo = &types.Tenant{ID: agentTenantID}
	}

	// Ensure defaults are set
	req.CustomAgent.EnsureDefaults()

	// Build AgentConfig from custom agent and tenant info
	agentConfig, err := s.buildAgentConfig(ctx, req, tenantInfo, agentTenantID)
	if err != nil {
		return err
	}

	// Set VLM model ID for tool result image analysis (runtime-only field)
	if req.CustomAgent != nil && req.CustomAgent.Config.VLMModelID != "" {
		agentConfig.VLMModelID = req.CustomAgent.Config.VLMModelID
	}

	// Resolve model ID using shared helper (AgentQA requires a model, so error if not found)
	effectiveModelID, err := s.resolveChatModelID(ctx, req, agentConfig.KnowledgeBases, agentConfig.KnowledgeIDs)
	if err != nil {
		return err
	}
	if effectiveModelID == "" {
		logger.Warnf(ctx, "No summary model configured for custom agent %s", req.CustomAgent.ID)
		return errors.New("summary model (model_id) is not configured in custom agent settings")
	}

	summaryModel, err := s.modelService.GetChatModel(ctx, effectiveModelID)
	if err != nil {
		logger.Warnf(ctx, "Failed to get chat model: %v", err)
		return fmt.Errorf("failed to get chat model: %w", err)
	}

	// Get rerank model from custom agent config (only required when knowledge bases are configured)
	var rerankModel rerank.Reranker
	hasKnowledge := len(agentConfig.KnowledgeBases) > 0 || len(agentConfig.KnowledgeIDs) > 0
	if hasKnowledge {
		rerankModelID := req.CustomAgent.Config.RerankModelID
		if rerankModelID == "" {
			logger.Warnf(ctx, "No rerank model configured for custom agent %s, but knowledge bases are specified", req.CustomAgent.ID)
			return errors.New("rerank model (rerank_model_id) is not configured in custom agent settings")
		}

		rerankModel, err = s.modelService.GetRerankModel(ctx, rerankModelID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get rerank model: %v", err)
			return fmt.Errorf("failed to get rerank model: %w", err)
		}
	} else {
		logger.Infof(ctx, "No knowledge bases configured, skipping rerank model initialization")
	}

	// Get or create contextManager for this session
	contextManager := s.getContextManagerForSession()

	// Set system prompt for the current agent in context manager
	// This ensures the context uses the correct system prompt when switching agents
	systemPrompt := agentConfig.ResolveSystemPrompt(agentConfig.WebSearchEnabled)
	if systemPrompt != "" {
		if err := contextManager.SetSystemPrompt(ctx, sessionID, systemPrompt); err != nil {
			logger.Warnf(ctx, "Failed to set system prompt in context manager: %v", err)
		} else {
			logger.Infof(ctx, "System prompt updated in context manager for agent")
		}
	}

	// Get LLM context from context manager
	llmContext, err := s.getContextForSession(ctx, contextManager, sessionID)
	if err != nil {
		logger.Warnf(ctx, "Failed to get LLM context: %v, continuing without history", err)
		llmContext = []chat.Message{}
	}
	logger.Infof(ctx, "Loaded %d messages from LLM context manager", len(llmContext))

	// Apply multi-turn configuration for Agent mode
	// Note: In Agent mode, context is managed by contextManager with compression strategies,
	// so we don't apply HistoryTurns limit here. HistoryTurns is used in normal (KnowledgeQA) mode.
	if !agentConfig.MultiTurnEnabled {
		// Multi-turn disabled, clear history
		logger.Infof(ctx, "Multi-turn disabled for this agent, clearing history context")
		llmContext = []chat.Message{}
	}

	// Create agent engine with EventBus and ContextManager
	logger.Info(ctx, "Creating agent engine")

	// Execute intent explore before creating the engine if enabled.
	// Rules:
	//   - Only execute for "new topics" (no or minimal history context).
	//   - For follow-up questions, skip to avoid misleading the agent.
	//   - Once IntentExploreQueries is set, it's consumed and cleared by KnowledgeSearchTool
	//     after the first-round call, so it won't affect subsequent rounds.
	enableIntentExplore := s.cfg.Conversation.EnableQueryIntentExplore
	if req.CustomAgent.Config.EnableQueryIntentExplore != nil {
		enableIntentExplore = *req.CustomAgent.Config.EnableQueryIntentExplore
	}
	if enableIntentExplore {
		// Determine if this is a "new topic" or follow-up question.
		// Heuristic: if the LLM context (history) has <= 1 messages, treat as new topic.
		isNewTopic := len(llmContext) <= 1
		if isNewTopic {
			intentData := s.executeIntentExplore(ctx, req.Query, summaryModel, eventBus, sessionID)
			if intentData != nil && len(intentData.FinalSearchQueries) > 0 {
				agentConfig.IntentExploreSystemBlock = formatIntentExploreSystemBlock(intentData)
				agentConfig.IntentExploreQueries = intentData.FinalSearchQueries
				logger.Infof(ctx, "Intent explore completed (new topic): %d paths, %d queries",
					len(intentData.AnalysisPaths), len(intentData.FinalSearchQueries))
			}
		} else {
			logger.Infof(ctx, "Intent explore skipped (follow-up question, history=%d messages)", len(llmContext))
		}
	}
	if enableIntentExplore {
		intentData := s.executeIntentExplore(ctx, req.Query, summaryModel, eventBus, sessionID)
		if intentData != nil && len(intentData.FinalSearchQueries) > 0 {
			agentConfig.IntentExploreSystemBlock = formatIntentExploreSystemBlock(intentData)
			agentConfig.IntentExploreQueries = intentData.FinalSearchQueries
			logger.Infof(ctx, "Intent explore completed: %d paths, %d search queries",
				len(intentData.AnalysisPaths), len(intentData.FinalSearchQueries))
		}
	}

	engine, err := s.agentService.CreateAgentEngine(
		ctx,
		agentConfig,
		summaryModel,
		rerankModel,
		eventBus,
		contextManager,
		sessionID,
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to create agent engine: %v", err)
		return err
	}

	// Route image data based on agent model's vision capability
	var agentModelSupportsVision bool
	if effectiveModelID != "" {
		if modelInfo, err := s.modelService.GetModelByID(ctx, effectiveModelID); err == nil && modelInfo != nil {
			agentModelSupportsVision = modelInfo.Parameters.SupportsVision
		}
	}

	agentQuery := req.Query
	var agentImageURLs []string
	if agentModelSupportsVision && len(req.ImageURLs) > 0 {
		agentImageURLs = req.ImageURLs
		logger.Infof(ctx, "Agent model supports vision, passing %d image(s) directly", len(agentImageURLs))
	} else if req.ImageDescription != "" {
		agentQuery = req.Query + "\n\n[用户上传图片内容]\n" + req.ImageDescription
		logger.Infof(ctx, "Agent model does not support vision, appending image description (%d chars)", len(req.ImageDescription))
	}
	if req.QuotedContext != "" {
		agentQuery += "\n\n" + req.QuotedContext
	}

	// Execute agent with streaming (asynchronously)
	// Events will be emitted to EventBus and handled by the Handler layer
	logger.Info(ctx, "Executing agent with streaming")
	if _, err := engine.Execute(ctx, sessionID, req.AssistantMessageID, agentQuery, llmContext, agentImageURLs); err != nil {
		logger.Errorf(ctx, "Agent execution failed: %v", err)
		// Emit error event to the EventBus used by this agent
		eventBus.Emit(ctx, event.Event{
			Type:      event.EventError,
			SessionID: sessionID,
			Data: event.ErrorData{
				Error:     err.Error(),
				Stage:     "agent_execution",
				SessionID: sessionID,
			},
		})
	}
	// Return empty - events will be handled by Handler via EventBus subscription
	return nil
}

// buildAgentConfig creates a runtime AgentConfig from the QARequest's custom agent configuration,
// tenant info, and resolved knowledge bases / search targets.
func (s *sessionService) buildAgentConfig(
	ctx context.Context,
	req *types.QARequest,
	tenantInfo *types.Tenant,
	agentTenantID uint64,
) (*types.AgentConfig, error) {
	customAgent := req.CustomAgent
	agentConfig := &types.AgentConfig{
		MaxIterations:               customAgent.Config.MaxIterations,
		Temperature:                 customAgent.Config.Temperature,
		WebSearchEnabled:            customAgent.Config.WebSearchEnabled && req.WebSearchEnabled,
		WebSearchMaxResults:         customAgent.Config.WebSearchMaxResults,
		WebSearchProviderID:         customAgent.Config.WebSearchProviderID,
		MultiTurnEnabled:            customAgent.Config.MultiTurnEnabled,
		HistoryTurns:                customAgent.Config.HistoryTurns,
		MCPSelectionMode:            customAgent.Config.MCPSelectionMode,
		MCPServices:                 customAgent.Config.MCPServices,
		Thinking:                    customAgent.Config.Thinking,
		RetrieveKBOnlyWhenMentioned: customAgent.Config.RetrieveKBOnlyWhenMentioned,
	}

	// Configure skills based on CustomAgentConfig
	s.configureSkillsFromAgent(ctx, agentConfig, customAgent)

	// Resolve knowledge bases using shared helper
	agentConfig.KnowledgeBases, agentConfig.KnowledgeIDs = s.resolveKnowledgeBases(ctx, req)

	// Use custom agent's allowed tools if specified, otherwise use defaults
	if len(customAgent.Config.AllowedTools) > 0 {
		agentConfig.AllowedTools = customAgent.Config.AllowedTools
	} else {
		agentConfig.AllowedTools = tools.DefaultAllowedTools()
	}

	// Use custom agent's system prompt if specified
	if customAgent.Config.SystemPrompt != "" {
		agentConfig.UseCustomSystemPrompt = true
		agentConfig.SystemPrompt = customAgent.Config.SystemPrompt
	}

	logger.Infof(ctx, "Custom agent config applied: MaxIterations=%d, Temperature=%.2f, AllowedTools=%v, WebSearchEnabled=%v",
		agentConfig.MaxIterations, agentConfig.Temperature, agentConfig.AllowedTools, agentConfig.WebSearchEnabled)

	// Set web search max results from tenant config if not set (default: 5)
	if agentConfig.WebSearchMaxResults == 0 {
		agentConfig.WebSearchMaxResults = 5
		if tenantInfo.WebSearchConfig != nil && tenantInfo.WebSearchConfig.MaxResults > 0 {
			agentConfig.WebSearchMaxResults = tenantInfo.WebSearchConfig.MaxResults
		}
	}

	// Resolve web search provider ID: agent-level > tenant default (is_default=true)
	if agentConfig.WebSearchProviderID == "" {
		if defaultProvider, err := s.webSearchProviderRepo.GetDefault(ctx, tenantInfo.ID); err == nil && defaultProvider != nil {
			agentConfig.WebSearchProviderID = defaultProvider.ID
		}
	}

	logger.Infof(ctx, "Merged agent config from tenant %d and session %s", tenantInfo.ID, req.Session.ID)

	// Log knowledge bases if present
	if len(agentConfig.KnowledgeBases) > 0 {
		logger.Infof(ctx, "Agent configured with %d knowledge base(s): %v",
			len(agentConfig.KnowledgeBases), agentConfig.KnowledgeBases)
	} else {
		logger.Infof(ctx, "No knowledge bases specified for agent, running in pure agent mode")
	}

	// Build search targets using agent's tenant (handler has validated access for shared agent)
	searchTargets, err := s.buildSearchTargets(ctx, agentTenantID, agentConfig.KnowledgeBases, agentConfig.KnowledgeIDs)
	if err != nil {
		logger.Warnf(ctx, "Failed to build search targets for agent: %v", err)
	}
	agentConfig.SearchTargets = searchTargets
	logger.Infof(ctx, "Agent search targets built: %d targets", len(searchTargets))

	if agentConfig.MaxContextTokens <= 0 {
		agentConfig.MaxContextTokens = types.DefaultMaxContextTokens
	}

	return agentConfig, nil
}

// configureSkillsFromAgent configures skills settings in AgentConfig based on CustomAgentConfig
// Returns the skill directories and allowed skills based on the selection mode:
//   - "all": uses all preloaded skills
//   - "selected": uses the explicitly selected skills
//   - "none" or "": skills are disabled
func (s *sessionService) configureSkillsFromAgent(
	ctx context.Context,
	agentConfig *types.AgentConfig,
	customAgent *types.CustomAgent,
) {
	if customAgent == nil {
		return
	}
	// When sandbox is disabled, skills cannot be enabled (no script execution environment)
	sandboxMode := os.Getenv("WEKNORA_SANDBOX_MODE")
	if sandboxMode == "" || sandboxMode == "disabled" {
		agentConfig.SkillsEnabled = false
		agentConfig.SkillDirs = nil
		agentConfig.AllowedSkills = nil
		logger.Infof(ctx, "Sandbox is disabled: skills are not available")
		return
	}

	switch customAgent.Config.SkillsSelectionMode {
	case "all":
		// Enable all preloaded skills
		agentConfig.SkillsEnabled = true
		agentConfig.SkillDirs = []string{DefaultPreloadedSkillsDir}
		agentConfig.AllowedSkills = nil // Empty means all skills allowed
		logger.Infof(ctx, "SkillsSelectionMode=all: enabled all preloaded skills")
	case "selected":
		// Enable only selected skills
		if len(customAgent.Config.SelectedSkills) > 0 {
			agentConfig.SkillsEnabled = true
			agentConfig.SkillDirs = []string{DefaultPreloadedSkillsDir}
			agentConfig.AllowedSkills = customAgent.Config.SelectedSkills
			logger.Infof(ctx, "SkillsSelectionMode=selected: enabled %d selected skills: %v",
				len(customAgent.Config.SelectedSkills), customAgent.Config.SelectedSkills)
		} else {
			agentConfig.SkillsEnabled = false
			logger.Infof(ctx, "SkillsSelectionMode=selected but no skills selected: skills disabled")
		}
	case "none", "":
		// Skills disabled
		agentConfig.SkillsEnabled = false
		logger.Infof(ctx, "SkillsSelectionMode=%s: skills disabled", customAgent.Config.SkillsSelectionMode)
	default:
		// Unknown mode, disable skills
		agentConfig.SkillsEnabled = false
		logger.Warnf(ctx, "Unknown SkillsSelectionMode=%s: skills disabled", customAgent.Config.SkillsSelectionMode)
	}

}

// getContextManagerForSession creates a context manager for the session.
func (s *sessionService) getContextManagerForSession() interfaces.ContextManager {
	return llmcontext.NewContextManagerFromConfig(s.sessionStorage, s.messageRepo)
}

// getContextForSession retrieves LLM context for a session
func (s *sessionService) getContextForSession(
	ctx context.Context,
	contextManager interfaces.ContextManager,
	sessionID string,
) ([]chat.Message, error) {
	history, err := contextManager.GetContext(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	// Log context statistics
	stats, _ := contextManager.GetContextStats(ctx, sessionID)
	if stats != nil {
		logger.Infof(ctx, "LLM context stats for session %s: messages=%d, tokens=~%d, compressed=%v",
			sessionID, stats.MessageCount, stats.TokenCount, stats.IsCompressed)
	}

	return history, nil
}

// executeIntentExplore performs pre-search intent exploration before the agent engine starts.
// It uses the intent explore prompt to decompose the user query into multiple search paths.
// The resulting queries are injected into the system prompt to guide the agent's
// first-round knowledge_search call, NOT executed here.
func (s *sessionService) executeIntentExplore(
	ctx context.Context,
	query string,
	chatModel chat.Chat,
	eventBus *event.EventBus,
	sessionID string,
) *types.IntentExploreData {
	// 1. Read prompt config
	promptContent := s.cfg.Conversation.IntentExplorePrompt
	if promptContent == "" {
		logger.Infof(ctx, "[IntentExplore] No prompt configured, skipping")
		return nil
	}
	userContent := s.cfg.Conversation.IntentExplorePromptUser
	if userContent == "" {
		userContent = query
	} else {
		userContent = strings.ReplaceAll(userContent, "{{query}}", query)
	}

	// 2. Call LLM to decompose query
	messages := []chat.Message{
		{Role: "system", Content: promptContent},
		{Role: "user", Content: userContent},
	}
	opt := &chat.ChatOptions{
		Temperature:         0.3,
		MaxCompletionTokens: 65536,
	}

	responseChan, err := chatModel.ChatStream(ctx, messages, opt)
	if err != nil {
		logger.Warnf(ctx, "[IntentExplore] Failed to start chat stream: %v", err)
		return nil
	}

	var fullContent strings.Builder
	for response := range responseChan {
		switch response.ResponseType {
		case types.ResponseTypeAnswer:
			fullContent.WriteString(response.Content)
		case types.ResponseTypeError:
			logger.Warnf(ctx, "[IntentExplore] Stream error: %s", response.Content)
			return nil
		}
	}

	fullContentString := fullContent.String()
	logger.Infof(ctx, "[IntentExplore] LLM response: %d chars", len(fullContentString))

	// 3. Parse output
	output := chatpipeline.ParseIntentExploreOutput(fullContentString)
	if output == nil || len(output.FinalSearchQueries) == 0 {
		logger.Warnf(ctx, "[IntentExplore] Failed to parse output or no search queries")
		return nil
	}

	// 4. Build IntentExploreData (no search executed here)
	intentData := &types.IntentExploreData{
		OriginalQuery:      query,
		AnalysisPaths:      make([]*types.AnalysisPath, 0, len(output.AnalysisPaths)),
		FinalSearchQueries: output.FinalSearchQueries,
		TotalSearchCount:   0,
	}
	for _, path := range output.AnalysisPaths {
		intentData.AnalysisPaths = append(intentData.AnalysisPaths, &types.AnalysisPath{
			PathID:               path.PathID,
			Entity:               path.Entity,
			Dimensions:           path.Dimensions,
			MergedSearchString:   path.MergedSearchString,
			Reason:               path.Reason,
			SourceEntity:         path.SourceEntity,
			TargetEntity:         path.TargetEntity,
			InteractionType:      path.InteractionType,
			MechanisticLink:      path.MechanisticLink,
			ClinicalSignificance: path.ClinicalSignificance,
		})
	}

	// 5. Emit EventQueryIntentExplore event (for frontend display)
	if eventBus != nil {
		paths := make([]*event.AnalysisPath, len(output.AnalysisPaths))
		for i, path := range output.AnalysisPaths {
			paths[i] = &event.AnalysisPath{
				PathID:               path.PathID,
				Entity:               path.Entity,
				Dimensions:           path.Dimensions,
				MergedSearchString:   path.MergedSearchString,
				Reason:               path.Reason,
				SourceEntity:         path.SourceEntity,
				TargetEntity:         path.TargetEntity,
				InteractionType:      path.InteractionType,
				MechanisticLink:      path.MechanisticLink,
				ClinicalSignificance: path.ClinicalSignificance,
			}
		}
		eventBus.Emit(ctx, event.Event{
			Type:      event.EventQueryIntentExplore,
			SessionID: sessionID,
			Data: event.QueryIntentExploreData{
				OriginalQuery:      query,
				AnalysisPaths:      paths,
				FinalSearchQueries: output.FinalSearchQueries,
				TotalSearchCount:   0, // No search executed here
			},
		})
	}

	logger.Infof(ctx, "[IntentExplore] Completed: %d paths, %d queries",
		len(output.AnalysisPaths), len(output.FinalSearchQueries))

	return intentData
}

// formatIntentExploreSystemBlock generates the intent analysis block for system prompt injection.
// The queries are explicitly listed so the LLM can use them in knowledge_search calls.
func formatIntentExploreSystemBlock(data *types.IntentExploreData) string {
	var sb strings.Builder
	sb.WriteString("## Intent Explore Analysis\n\n")
	sb.WriteString("The user's query has been pre-analyzed with the following intent structure:\n\n")
	sb.WriteString(fmt.Sprintf("Original Query: %s\n\n", data.OriginalQuery))
	sb.WriteString("### Analysis Paths\n")
	sb.WriteString("| Path | Entity | Dimensions | Search Strategy |\n")
	sb.WriteString("|------|--------|-----------|-----------------|\n")
	for _, path := range data.AnalysisPaths {
		dims := strings.Join(path.Dimensions, ", ")
		if dims == "" {
			dims = "-"
		}
		searchStr := path.MergedSearchString
		if searchStr == "" && path.SourceEntity != "" {
			searchStr = fmt.Sprintf("%s ↔ %s (%s)", path.SourceEntity, path.TargetEntity, path.InteractionType)
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", path.PathID, path.Entity, dims, searchStr))
	}
	sb.WriteString(fmt.Sprintf("\n### Pre-analyzed Search Queries\n"))
	sb.WriteString("The following queries have been pre-analyzed from the user's intent.\n")
	sb.WriteString("**You MUST use these queries in your first-round knowledge_search call.**\n\n")
	for i, q := range data.FinalSearchQueries {
		sb.WriteString(fmt.Sprintf("%d. \"%s\"\n", i+1, q))
	}
	sb.WriteString("\n### Guidance\n")
	sb.WriteString("- Use the above queries as the `queries` parameter in your first knowledge_search call.\n")
	sb.WriteString("- Do NOT generate your own queries - use the pre-analyzed ones above.\n")
	sb.WriteString("- If the pre-analyzed queries are insufficient, you may make additional knowledge_search calls.\n")
	return sb.String()
}
