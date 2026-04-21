package chatpipeline

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type PluginQueryIntentExplore struct {
	modelService interfaces.ModelService
	config       *config.Config
	searchPlugin *PluginSearch
}

type intentExploreOutput struct {
	OriginalQuery      string         `json:"original_query"`
	AnalysisPaths      []analysisPath `json:"analysis_paths"`
	FinalSearchQueries []string       `json:"final_search_queries"`
}

type analysisPath struct {
	PathID             int      `json:"path_id"`
	Entity             string   `json:"entity"`
	Dimensions         []string `json:"dimensions"`
	MergedSearchString string   `json:"merged_search_string"`
	Reason             string   `json:"reason"`
}

func NewPluginQueryIntentExplore(
	eventManager *EventManager,
	modelService interfaces.ModelService,
	config *config.Config,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	chunkService interfaces.ChunkService,
	webSearchService interfaces.WebSearchService,
	tenantService interfaces.TenantService,
	sessionService interfaces.SessionService,
	webSearchStateService interfaces.WebSearchStateService,
	webSearchProviderRepo interfaces.WebSearchProviderRepository,
) *PluginQueryIntentExplore {
	searchPlugin := &PluginSearch{
		knowledgeBaseService:  knowledgeBaseService,
		knowledgeService:      knowledgeService,
		chunkService:          chunkService,
		config:                config,
		webSearchService:      webSearchService,
		tenantService:         tenantService,
		sessionService:        sessionService,
		webSearchStateService: webSearchStateService,
		webSearchProviderRepo: webSearchProviderRepo,
	}

	res := &PluginQueryIntentExplore{
		modelService: modelService,
		config:       config,
		searchPlugin: searchPlugin,
	}
	eventManager.Register(res)
	return res
}

func (p *PluginQueryIntentExplore) ActivationEvents() []types.EventType {
	return []types.EventType{types.QUERY_INTENT_EXPLORE}
}

func (p *PluginQueryIntentExplore) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	if !chatManage.EnableQueryIntentExplore {
		pipelineInfo(ctx, "QueryIntentExplore", "skip", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     "feature_disabled",
		})
		return next()
	}

	pipelineInfo(ctx, "QueryIntentExplore", "start", map[string]interface{}{
		"session_id":    chatManage.SessionID,
		"rewrite_query": chatManage.RewriteQuery,
	})

	model, err := p.modelService.GetChatModel(ctx, chatManage.ChatModelID)
	if err != nil {
		pipelineError(ctx, "QueryIntentExplore", "get_model", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return next()
	}

	promptContent := p.config.Conversation.IntentExplorePrompt
	if promptContent == "" {
		pipelineWarn(ctx, "QueryIntentExplore", "no_prompt", map[string]interface{}{
			"session_id":       chatManage.SessionID,
			"config_prompt_id": chatManage.IntentExplorePromptID,
		})
		return next()
	}
	userContent := p.config.Conversation.IntentExplorePromptUser
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
		MaxCompletionTokens: 1500,
	}
	resp, err := model.Chat(ctx, messages, opt)
	if err != nil {
		pipelineError(ctx, "QueryIntentExplore", "model_call", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"error":      err.Error(),
		})
		return next()
	}

	output := p.parseOutput(resp.Content)
	if output == nil {
		pipelineWarn(ctx, "QueryIntentExplore", "parse_failed", map[string]interface{}{
			"session_id":   chatManage.SessionID,
			"raw_response": resp.Content,
		})
		return next()
	}

	chatManage.IntentExploreData = &types.IntentExploreData{
		OriginalQuery:      chatManage.RewriteQuery,
		FinalSearchQueries: output.FinalSearchQueries,
	}
	for _, path := range output.AnalysisPaths {
		chatManage.IntentExploreData.AnalysisPaths = append(chatManage.IntentExploreData.AnalysisPaths, &types.AnalysisPath{
			PathID:             path.PathID,
			Entity:             path.Entity,
			Dimensions:         path.Dimensions,
			MergedSearchString: path.MergedSearchString,
			Reason:             path.Reason,
		})
	}

	pipelineInfo(ctx, "QueryIntentExplore", "intent_explored", map[string]interface{}{
		"session_id":        chatManage.SessionID,
		"path_count":        len(output.AnalysisPaths),
		"final_query_count": len(output.FinalSearchQueries),
	})

	p.searchMultiplePaths(ctx, chatManage, output.FinalSearchQueries)

	if chatManage.EventBus != nil {
		paths := make([]*event.AnalysisPath, len(output.AnalysisPaths))
		for i, path := range output.AnalysisPaths {
			paths[i] = &event.AnalysisPath{
				PathID:             path.PathID,
				Entity:             path.Entity,
				Dimensions:         path.Dimensions,
				MergedSearchString: path.MergedSearchString,
				Reason:             path.Reason,
			}
		}
		chatManage.EventBus.Emit(ctx, types.Event{
			Type:      types.EventType(event.EventQueryIntentExplore),
			SessionID: chatManage.SessionID,
			Data: event.QueryIntentExploreData{
				OriginalQuery:      chatManage.RewriteQuery,
				AnalysisPaths:      paths,
				FinalSearchQueries: output.FinalSearchQueries,
				TotalSearchCount:   len(chatManage.SearchResult),
			}})
	}

	return next()
}

func (p *PluginQueryIntentExplore) searchMultiplePaths(ctx context.Context,
	chatManage *types.ChatManage, queries []string,
) {
	if len(queries) == 0 {
		return
	}

	var mu sync.Mutex
	allResults := make([]*types.SearchResult, 0)
	var wg sync.WaitGroup
	wg.Add(len(queries))

	for _, query := range queries {
		go func(q string) {
			defer wg.Done()
			results := p.searchSinglePath(ctx, chatManage, q)
			if len(results) > 0 {
				mu.Lock()
				allResults = append(allResults, results...)
				mu.Unlock()
			}
		}(query)
	}

	wg.Wait()

	chatManage.SearchResult = allResults

	pipelineInfo(ctx, "QueryIntentExplore", "multi_search_done", map[string]interface{}{
		"session_id":   chatManage.SessionID,
		"query_count":  len(queries),
		"result_count": len(allResults),
	})
}

func (p *PluginQueryIntentExplore) searchSinglePath(ctx context.Context,
	chatManage *types.ChatManage, query string,
) []*types.SearchResult {
	hasKBTargets := len(chatManage.SearchTargets) > 0 || len(chatManage.KnowledgeBaseIDs) > 0 || len(chatManage.KnowledgeIDs) > 0
	if !hasKBTargets && !chatManage.WebSearchEnabled {
		return nil
	}

	searchCM := chatManage.Clone()
	searchCM.RewriteQuery = query
	searchCM.SearchResult = nil

	noop := func() *PluginError { return nil }

	err := p.searchPlugin.OnEvent(ctx, types.CHUNK_SEARCH, searchCM, noop)
	if err != nil || len(searchCM.SearchResult) == 0 {
		if err != nil {
			logger.Debugf(ctx, "Single path search failed for query %s: %v", query, err)
		}
		return nil
	}

	return searchCM.SearchResult
}

func (p *PluginQueryIntentExplore) parseOutput(content string) *intentExploreOutput {
	content = strings.TrimSpace(content)
	logger.Infof(context.Background(), "IntentExplore: raw response: %s", content)
	if content == "" {
		return nil
	}

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		logger.Debugf(context.Background(), "IntentExplore: no JSON found in response")
		return nil
	}

	content = content[start : end+1]

	var out intentExploreOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		logger.Debugf(context.Background(), "IntentExplore: JSON parse error: %v", err)
		return nil
	}

	if len(out.FinalSearchQueries) == 0 {
		logger.Debugf(context.Background(), "IntentExplore: no final_search_queries")
		return nil
	}

	return &out
}
