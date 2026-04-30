package chatpipeline

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/redis/go-redis/v9"
)

type PluginQueryIntentExplore struct {
	modelService interfaces.ModelService
	config       *config.Config
	searchPlugin *PluginSearch
	redisClient  *redis.Client
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
	// Relation-type path fields
	SourceEntity         string `json:"source_entity"`
	TargetEntity         string `json:"target_entity"`
	InteractionType      string `json:"interaction_type"`
	MechanisticLink      string `json:"mechanistic_link"`
	ClinicalSignificance string `json:"clinical_significance"`
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
	redisClient *redis.Client,
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
		redisClient:  redisClient,
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
	if !chatManage.EnableQueryIntentExplore || chatManage.SkipIntentExplore {
		pipelineInfo(ctx, "QueryIntentExplore", "skip", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     "feature_disabled_or_domain_check_failed",
		})
		return next()
	}

	pipelineInfo(ctx, "QueryIntentExplore", "start", map[string]interface{}{
		"session_id":    chatManage.SessionID,
		"rewrite_query": chatManage.RewriteQuery,
	})

	// Try to get result from cache first
	cacheKey := p.getCacheKey(chatManage.Query)
	logger.Infof(ctx, "QueryIntentExplore Cache key: %s", cacheKey)
	cachedOutput, err := p.getFromCache(ctx, cacheKey)
	if err != nil {
		logger.Debugf(ctx, "Failed to get from cache: %v", err)
	}
	if cachedOutput != nil {
		pipelineInfo(ctx, "QueryIntentExplore", "cache_hit", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"cache_key":  cacheKey,
		})
		// Use cached output
		output := cachedOutput
		chatManage.IntentExploreData = &types.IntentExploreData{
			OriginalQuery:      chatManage.RewriteQuery,
			FinalSearchQueries: output.FinalSearchQueries,
		}
		for _, path := range output.AnalysisPaths {
			chatManage.IntentExploreData.AnalysisPaths = append(chatManage.IntentExploreData.AnalysisPaths, &types.AnalysisPath{
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

		pipelineInfo(ctx, "QueryIntentExplore", "intent_explored", map[string]interface{}{
			"session_id":        chatManage.SessionID,
			"path_count":        len(output.AnalysisPaths),
			"final_query_count": len(output.FinalSearchQueries),
		})

		if chatManage.EventBus != nil {
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
		// 优先触发sse事件，再触发搜索
		p.searchMultiplePaths(ctx, chatManage, output.FinalSearchQueries)

		return next()
	}

	modelID := chatManage.ChatModelID
	if chatManage.IntentExploreModelID != "" {
		modelID = chatManage.IntentExploreModelID
	}
	model, err := p.modelService.GetChatModel(ctx, modelID)
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
		MaxCompletionTokens: 65536,
	}
	// 使用chatStream流失式输出，实时解析JSON数据
	// Use streaming to get the model response
	responseChan, err := model.ChatStream(ctx, messages, opt)
	if err != nil {
		logger.Errorf(ctx, "failed to start chat stream: %v", err)
		return next()
	}

	// Collect all answer chunks from the stream
	var fullContent strings.Builder
	var streamErr error
	var responseTypes []string
	for response := range responseChan {
		responseTypes = append(responseTypes, string(response.ResponseType))
		switch response.ResponseType {
		case types.ResponseTypeAnswer:
			fullContent.WriteString(response.Content)
		case types.ResponseTypeError:
			logger.Errorf(ctx, "stream error: %s", response.Content)
			streamErr = fmt.Errorf("stream error: %s", response.Content)
		case types.ResponseTypeThinking:
			logger.Debugf(ctx, "QueryIntentExplore thinking: %s", response.Content)
		case types.ResponseTypeComplete:
			logger.Debugf(ctx, "[QueryIntentExplore] complete received")
		default:
			logger.Debugf(ctx, "[QueryIntentExplore] unhandled response type: %s, content: %s", response.ResponseType, response.Content)
		}
	}
	// logger.Warnf(ctx, "[Extract] response types collected: %v", responseTypes)
	if streamErr != nil {
		return next()
	}

	fullContentString := fullContent.String()

	logger.Debugf(ctx, "QueryIntentExplore content llm response: %s", fullContentString)

	output := p.parseOutput(fullContentString)
	if output == nil {
		pipelineWarn(ctx, "QueryIntentExplore", "parse_failed", map[string]interface{}{
			"session_id":   chatManage.SessionID,
			"raw_response": fullContentString,
		})
		return next()
	}

	// Cache the result for future use (only if output is not empty)
	if len(output.FinalSearchQueries) > 0 {
		if err := p.setToCache(ctx, cacheKey, output, 24*time.Hour); err != nil {
			logger.Warnf(ctx, "Failed to cache intent explore result: %v", err)
		} else {
			pipelineInfo(ctx, "QueryIntentExplore", "cached", map[string]interface{}{
				"session_id": chatManage.SessionID,
				"cache_key":  cacheKey,
			})
		}
	} else {
		pipelineInfo(ctx, "QueryIntentExplore", "skip_cache", map[string]interface{}{
			"session_id": chatManage.SessionID,
			"reason":     "empty_output",
		})
	}

	chatManage.IntentExploreData = &types.IntentExploreData{
		OriginalQuery:      chatManage.RewriteQuery,
		FinalSearchQueries: output.FinalSearchQueries,
	}
	for _, path := range output.AnalysisPaths {
		chatManage.IntentExploreData.AnalysisPaths = append(chatManage.IntentExploreData.AnalysisPaths, &types.AnalysisPath{
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

	pipelineInfo(ctx, "QueryIntentExplore", "intent_explored", map[string]interface{}{
		"session_id":        chatManage.SessionID,
		"path_count":        len(output.AnalysisPaths),
		"final_query_count": len(output.FinalSearchQueries),
	})

	if chatManage.EventBus != nil {
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
	// 优先触发sse事件，再触发搜索
	p.searchMultiplePaths(ctx, chatManage, output.FinalSearchQueries)

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

	chatManage.SearchResult = removeDuplicateResults(append(chatManage.SearchResult, allResults...))

	pipelineInfo(ctx, "QueryIntentExplore", "multi_search_done", map[string]interface{}{
		"session_id":   chatManage.SessionID,
		"query_count":  len(queries),
		"result_count": len(chatManage.SearchResult),
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
	searchCM.SearchResult = removeDuplicateResults(searchCM.SearchResult)
	return searchCM.SearchResult
}

func (p *PluginQueryIntentExplore) parseOutput(content string) *intentExploreOutput {
	return ParseIntentExploreOutput(content)
}

// IntentExploreOutput is the exported version of intentExploreOutput for use by other packages.
type IntentExploreOutput = intentExploreOutput

// IntentExploreAnalysisPath is the exported version of analysisPath for use by other packages.
type IntentExploreAnalysisPath = analysisPath

// ParseIntentExploreOutput extracts and parses the intent explore JSON from LLM response content.
// Returns nil if the content is empty or the JSON cannot be parsed.
func ParseIntentExploreOutput(content string) *IntentExploreOutput {
	content = strings.TrimSpace(content)
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

	var out IntentExploreOutput
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

// getCacheKey generates a cache key for the given query
func (p *PluginQueryIntentExplore) getCacheKey(query string) string {
	hash := md5.Sum([]byte(query))
	return fmt.Sprintf("intent_explore:%x", hash)
}

// getFromCache retrieves intent explore output from cache
// Returns (nil, nil) if key not found (cache miss)
func (p *PluginQueryIntentExplore) getFromCache(ctx context.Context, key string) (*intentExploreOutput, error) {
	if p.redisClient == nil {
		logger.Warnf(ctx, "QueryIntentExplore: redis client not initialized")
		return nil, nil
	}

	data, err := p.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key not found - normal cache miss
			return nil, nil
		}
		return nil, err
	}

	var output intentExploreOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

// setToCache stores intent explore output in cache
func (p *PluginQueryIntentExplore) setToCache(ctx context.Context, key string, output *intentExploreOutput, expiration time.Duration) error {
	if p.redisClient == nil {
		logger.Warnf(ctx, "QueryIntentExplore: redis client not initialized")
		return nil
	}

	data, err := json.Marshal(output)
	if err != nil {
		return err
	}

	return p.redisClient.Set(ctx, key, data, expiration).Err()
}
