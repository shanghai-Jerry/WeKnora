package chatpipeline

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// PluginDomainCheck performs domain relevance checking before intent explore.
// It determines whether the user's query belongs to the ophthalmology domain.
type PluginDomainCheck struct {
	modelService interfaces.ModelService
	config       *config.Config
}

// domainCheckOutput represents the structured output from domain check LLM.
type domainCheckOutput struct {
	IsOphthalmology bool   `json:"is_ophthalmology"`
	Reason          string `json:"reason"`
}

// NewPluginDomainCheck creates a new domain check plugin instance
// and registers it with the event manager.
func NewPluginDomainCheck(
	eventManager *EventManager,
	modelService interfaces.ModelService,
	config *config.Config,
) *PluginDomainCheck {
	res := &PluginDomainCheck{
		modelService: modelService,
		config:       config,
	}
	eventManager.Register(res)
	return res
}

// ActivationEvents returns the list of event types this plugin responds to.
func (p *PluginDomainCheck) ActivationEvents() []types.EventType {
	return []types.EventType{types.DOMAIN_CHECK}
}

// OnEvent processes triggered events.
// It checks if the user's query belongs to the ophthalmology domain.
// If not, it sets SkipIntentExplore flag to true on the chatManage.
func (p *PluginDomainCheck) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	// Skip if domain check is disabled
	if !p.config.Conversation.EnableDomainCheck {
		pipelineInfo(ctx, "DomainCheck", "skip", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     "feature_disabled",
		})
		return next()
	}

	promptContent := p.config.Conversation.DomainCheckPrompt
	if promptContent == "" {
		pipelineWarn(ctx, "DomainCheck", "no_prompt", map[string]interface{}{
			"session_id": chatManage.SessionID,
		})
		return next()
	}

	pipelineInfo(ctx, "DomainCheck", "start", map[string]interface{}{
		"session_id":    chatManage.SessionID,
		"rewrite_query": chatManage.RewriteQuery,
	})

	// Select model for domain check
	modelID := chatManage.ChatModelID
	model, err := p.modelService.GetChatModel(ctx, modelID)
	if err != nil {
		pipelineError(ctx, "DomainCheck", "get_model", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return next()
	}

	userContent := p.config.Conversation.DomainCheckPromptUser
	if userContent == "" {
		userContent = chatManage.RewriteQuery
	} else {
		userContent = strings.ReplaceAll(userContent, "{{query}}", chatManage.RewriteQuery)
	}
	messages := []chat.Message{
		{
			Role: "system", Content: promptContent,
		},
		{
			Role: "user", Content: userContent,
		},
	}
	opt := &chat.ChatOptions{
		Temperature:         0.3,
		MaxCompletionTokens: 1024,
	}
	// Call model
	response, err := model.Chat(ctx, messages, opt)
	if err != nil {
		pipelineError(ctx, "DomainCheck", "model_call", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return next()
	}

	// Parse output
	output := p.parseOutput(response.Content)
	if output == nil {
		pipelineWarn(ctx, "DomainCheck", "parse_failed", map[string]interface{}{
			"session_id":   chatManage.SessionID,
			"raw_response": response.Content,
		})
		return next()
	}

	// Set skip flag if not ophthalmology domain
	skippedIntent := false
	if !output.IsOphthalmology {
		chatManage.SkipIntentExplore = true
		skippedIntent = true
		pipelineInfo(ctx, "DomainCheck", "skip_intent_explore", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     output.Reason,
		})
	} else {
		pipelineInfo(ctx, "DomainCheck", "is_ophthalmology", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     output.Reason,
		})
	}

	// Emit SSE event for frontend
	if chatManage.EventBus != nil {
		chatManage.EventBus.Emit(ctx, types.Event{
			Type:      types.EventType(event.EventDomainCheck),
			SessionID: chatManage.SessionID,
			Data: event.DomainCheckData{
				IsOphthalmology: output.IsOphthalmology,
				Reason:          output.Reason,
				SkippedIntent:   skippedIntent,
			},
		})
	}

	return next()
}

// parseOutput extracts and parses the domain check JSON from LLM response.
func (p *PluginDomainCheck) parseOutput(content string) *domainCheckOutput {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	// Try to find JSON in the response
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		logger.Debugf(context.Background(), "DomainCheck: no JSON found in response")
		return nil
	}

	candidate := content[start : end+1]

	var out domainCheckOutput
	if err := json.Unmarshal([]byte(candidate), &out); err != nil {
		logger.Debugf(context.Background(), "DomainCheck: JSON parse error: %v", err)
		return nil
	}

	return &out
}
