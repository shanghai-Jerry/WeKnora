package chatpipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

const (
	defaultRAGSystemPrompt = `You are a helpful knowledge assistant. Given the user's question and some reference information, provide an accurate answer.

You must respond in the following JSON format:
- If you can provide a complete answer based on the available information:
  {"action": "answer", "content": "your complete answer"}
- If you need more information to answer the question properly:
  {"action": "retrieve", "content": "your current understanding or partial answer", "query": "the search query to find more information"}

Guidelines:
- Only request retrieval when you genuinely lack sufficient information to answer.
- When requesting retrieval, provide a concise and specific search query.
- When answering, cite the reference sources when possible using [1], [2] etc.
- Respond in the same language as the user's question.`

	defaultRAGForceAnswerSystemPrompt = `You are a helpful knowledge assistant. Based on the user's question and the available reference information, you MUST provide your best possible answer now. You cannot request more information.

Respond in the following JSON format only:
{"action": "answer", "content": "your best complete answer based on available information"}

Guidelines:
- Provide the best answer you can with the information available.
- If the references are insufficient, state what you know and acknowledge limitations.
- Cite the reference sources when possible using [1], [2] etc.
- Respond in the same language as the user's question.`
)

type PluginRAGIterate struct {
	knowledgeBaseService interfaces.KnowledgeBaseService
	knowledgeService     interfaces.KnowledgeService
	modelService         interfaces.ModelService
}

func NewPluginRAGIterate(
	eventManager *EventManager,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	modelService interfaces.ModelService,
) *PluginRAGIterate {
	res := &PluginRAGIterate{
		knowledgeBaseService: knowledgeBaseService,
		knowledgeService:     knowledgeService,
		modelService:         modelService,
	}
	eventManager.Register(res)
	return res
}

func (p *PluginRAGIterate) ActivationEvents() []types.EventType {
	return []types.EventType{types.RAG_ITERATE}
}

func (p *PluginRAGIterate) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	pipelineInfo(ctx, "RAGIterate", "start", map[string]interface{}{
		"session_id":  chatManage.SessionID,
		"query":       chatManage.Query,
		"max_rounds":  chatManage.RAGMaxRounds,
		"has_kb":      len(chatManage.SearchTargets) > 0,
		"has_history": len(chatManage.History) > 0,
	})

	maxRounds := chatManage.RAGMaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}

	state := &types.RAGIterationState{
		MaxRounds:     maxRounds,
		AllReferences: make([]*types.SearchResult, 0),
		IterationSteps: make([]types.RAGIterationStep, 0),
	}
	chatManage.RAGIterationState = state

	eventBus := chatManage.EventBus

	referenceText := ""
	seenChunkIDs := make(map[string]bool)

	for round := 1; round <= maxRounds; round++ {
		state.CurrentRound = round
		isLastRound := round == maxRounds

		systemPrompt := chatManage.RAGRetrievalPrompt
		if systemPrompt == "" {
			if isLastRound {
				systemPrompt = defaultRAGForceAnswerSystemPrompt
			} else {
				systemPrompt = defaultRAGSystemPrompt
			}
		}

		userContent := buildRAGUserContent(chatManage.Query, state.Intermediary, referenceText, chatManage.Language)

		messages := []chat.Message{
			{Role: "system", Content: systemPrompt},
		}
		for _, h := range chatManage.History {
			messages = append(messages, chat.Message{Role: "user", Content: h.Query})
			messages = append(messages, chat.Message{Role: "assistant", Content: h.Answer})
		}
		messages = append(messages, chat.Message{Role: "user", Content: userContent})

		chatModel, opt, err := prepareChatModel(ctx, p.modelService, chatManage)
		if err != nil {
			return ErrGetChatModel.WithError(err)
		}

		pipelineInfo(ctx, "RAGIterate", "llm_call", map[string]interface{}{
			"round":       round,
			"model_id":    chatManage.ChatModelID,
			"is_last":     isLastRound,
		})

		response, err := chatModel.Chat(ctx, messages, opt)
		if err != nil {
			pipelineError(ctx, "RAGIterate", "llm_error", map[string]interface{}{
				"round": round,
				"error": err.Error(),
			})
			return ErrModelCall.WithError(err)
		}

		rawResponse := ""
		if response != nil {
			rawResponse = response.Content
		}

		pipelineInfo(ctx, "RAGIterate", "llm_response", map[string]interface{}{
			"round":         round,
			"response_len":  len(rawResponse),
		})

		action, content, retrieveQuery := parseRAGResponse(rawResponse)

		step := types.RAGIterationStep{
			Round:     round,
			LLMAction: action,
			Content:   content,
		}

		if eventBus != nil {
			iterationData := event.RAGIterationData{
				Round:   round,
				Action:  action,
				Content: content,
				Done:    false,
			}
			if action == "retrieve" {
				iterationData.RetrieveQuery = retrieveQuery
			}
			eventBus.Emit(ctx, types.Event{
				ID:        fmt.Sprintf("rag-iter-%s", uuid.New().String()[:8]),
				Type:      types.EventType(event.EventRAGIteration),
				SessionID: chatManage.SessionID,
				Data:      iterationData,
			})
		}

		if action == "answer" || isLastRound {
			if action == "retrieve" && isLastRound {
				state.Intermediary = content
				forceAnswer := tryForceAnswer(ctx, chatModel, opt, chatManage, state, referenceText)
				if forceAnswer != "" {
					content = forceAnswer
				}
			}

			state.FinalAnswer = content
			state.IsCompleted = true
			step.LLMAction = "answer"
			step.Content = content

			state.IterationSteps = append(state.IterationSteps, step)
			chatManage.MergeResult = state.AllReferences

			if eventBus != nil {
				eventBus.Emit(ctx, types.Event{
					ID:        fmt.Sprintf("rag-iter-%s", uuid.New().String()[:8]),
					Type:      types.EventType(event.EventRAGIteration),
					SessionID: chatManage.SessionID,
					Data: event.RAGIterationData{
						Round:   round,
						Action:  "answer",
						Content: content,
						Done:    true,
					},
				})
			}

			answerID := fmt.Sprintf("%s-answer", uuid.New().String()[:8])
			if eventBus != nil {
				eventBus.Emit(ctx, types.Event{
					ID:        answerID,
					Type:      types.EventType(event.EventAgentFinalAnswer),
					SessionID: chatManage.SessionID,
					Data: event.AgentFinalAnswerData{
						Content: content,
						Done:    false,
					},
				})
				eventBus.Emit(ctx, types.Event{
					ID:        answerID,
					Type:      types.EventType(event.EventAgentFinalAnswer),
					SessionID: chatManage.SessionID,
					Data: event.AgentFinalAnswerData{
						Content: "",
						Done:    true,
					},
				})
			}
			chatManage.ChatResponse = &types.ChatResponse{Content: content}

			pipelineInfo(ctx, "RAGIterate", "completed", map[string]interface{}{
				"round":           round,
				"total_refs":      len(state.AllReferences),
				"answer_len":      len(content),
			})
			return next()
		}

		state.Intermediary = content
		step.RetrieveQuery = retrieveQuery

		if retrieveQuery != "" && len(chatManage.SearchTargets) > 0 {
			newChunks, err := p.retrieveAndRerank(ctx, chatManage, retrieveQuery)
			if err != nil {
				pipelineWarn(ctx, "RAGIterate", "retrieve_error", map[string]interface{}{
					"round": round,
					"query": retrieveQuery,
					"error": err.Error(),
				})
			} else {
				var dedupedChunks []*types.SearchResult
				for _, chunk := range newChunks {
					if !seenChunkIDs[chunk.ID] {
						seenChunkIDs[chunk.ID] = true
						dedupedChunks = append(dedupedChunks, chunk)
						state.AllReferences = append(state.AllReferences, chunk)
					}
				}
				step.RetrievedChunks = dedupedChunks

				if len(dedupedChunks) > 0 {
					referenceText = buildReferenceText(appendAccumulatedRefText(referenceText, dedupedChunks))
				}

				pipelineInfo(ctx, "RAGIterate", "retrieve_result", map[string]interface{}{
					"round":        round,
					"query":        retrieveQuery,
					"new_chunks":   len(dedupedChunks),
					"total_refs":   len(state.AllReferences),
				})
			}
		}

		state.IterationSteps = append(state.IterationSteps, step)

		if eventBus != nil && len(step.RetrievedChunks) > 0 {
			eventBus.Emit(ctx, types.Event{
				ID:        fmt.Sprintf("rag-iter-%s", uuid.New().String()[:8]),
				Type:      types.EventType(event.EventRAGIteration),
				SessionID: chatManage.SessionID,
				Data: event.RAGIterationData{
					Round:        round,
					Action:       "retrieve",
					RetrieveQuery: retrieveQuery,
					ChunkCount:   len(step.RetrievedChunks),
					Done:         false,
				},
			})
		}
	}

	return next()
}

func (p *PluginRAGIterate) retrieveAndRerank(
	ctx context.Context,
	chatManage *types.ChatManage,
	query string,
) ([]*types.SearchResult, error) {
	searchTargets := chatManage.SearchTargets
	if len(searchTargets) == 0 {
		return nil, nil
	}

	var allResults []*types.SearchResult

	kbIDs := make([]string, 0, len(searchTargets))
	for _, t := range searchTargets {
		kbIDs = append(kbIDs, t.KnowledgeBaseID)
	}

	var fullKBIDs []string
	for _, t := range searchTargets {
		if t.Type == types.SearchTargetTypeKnowledgeBase {
			fullKBIDs = append(fullKBIDs, t.KnowledgeBaseID)
		}
	}

	if len(fullKBIDs) > 0 {
		params := types.SearchParams{
			QueryText:             query,
			KnowledgeBaseIDs:      fullKBIDs,
			VectorThreshold:       chatManage.VectorThreshold,
			KeywordThreshold:      chatManage.KeywordThreshold,
			MatchCount:            chatManage.EmbeddingTopK,
			SkipContextEnrichment: true,
		}
		res, err := p.knowledgeBaseService.HybridSearch(ctx, fullKBIDs[0], params)
		if err != nil {
			logger.Warnf(ctx, "RAG iterate HybridSearch failed: %v", err)
		} else {
			allResults = append(allResults, res...)
		}
	}

	if len(allResults) == 0 {
		return nil, nil
	}

	reranked, err := p.rerankResults(ctx, chatManage, query, allResults)
	if err != nil {
		logger.Warnf(ctx, "RAG iterate rerank failed, using raw results: %v", err)
		reranked = allResults
	}

	return reranked, nil
}

func (p *PluginRAGIterate) rerankResults(
	ctx context.Context,
	chatManage *types.ChatManage,
	query string,
	results []*types.SearchResult,
) ([]*types.SearchResult, error) {
	if chatManage.RerankModelID == "" || len(results) == 0 {
		return results, nil
	}

	rerankModel, err := p.modelService.GetRerankModel(ctx, chatManage.RerankModelID)
	if err != nil {
		return nil, err
	}

	var passages []string
	var candidates []*types.SearchResult

	for _, result := range results {
		if result.MatchType == types.MatchTypeDirectLoad {
			result.Score = 1.0
			continue
		}
		cleaned := cleanPassageForRerank(result.Content)
		enriched := getEnrichedPassage(ctx, result)
		if enriched != "" {
			cleaned = enriched
		} else if cleaned == "" {
			cleaned = result.Content
		}
		passages = append(passages, cleaned)
		candidates = append(candidates, result)
	}

	if len(passages) == 0 {
		return results, nil
	}

	batchSize := 20
	var allRanked []*types.SearchResult

	for i := 0; i < len(passages); i += batchSize {
		end := min(i+batchSize, len(passages))
		batchPassages := passages[i:end]
		batchCandidates := candidates[i:end]

		rankResults, err := rerankModel.Rerank(ctx, query, batchPassages)
		if err != nil {
			logger.Warnf(ctx, "RAG iterate rerank batch failed: %v", err)
			allRanked = append(allRanked, batchCandidates...)
			continue
		}

		for j, rr := range rankResults {
			if j < len(batchCandidates) {
				batchCandidates[j].RerankScore = rr.RelevanceScore
				batchCandidates[j].Score = rr.RelevanceScore
				allRanked = append(allRanked, batchCandidates[j])
			}
		}
	}

	slices.SortFunc(allRanked, func(a, b *types.SearchResult) int {
		if a.Score > b.Score {
			return -1
		} else if a.Score < b.Score {
			return 1
		}
		return 0
	})

	threshold := chatManage.RerankThreshold
	if threshold <= 0 {
		threshold = 0.3
	}

	var filtered []*types.SearchResult
	for _, r := range allRanked {
		if r.Score >= threshold {
			filtered = append(filtered, r)
		}
	}

	topK := chatManage.RerankTopK
	if topK <= 0 {
		topK = 5
	}
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}

	if len(filtered) == 0 && len(allRanked) > 0 {
		safeTop := min(topK, len(allRanked))
		filtered = allRanked[:safeTop]
	}

	return filtered, nil
}

type ragResponse struct {
	Action  string `json:"action"`
	Content string `json:"content"`
	Query   string `json:"query,omitempty"`
}

func parseRAGResponse(raw string) (action, content, query string) {
	raw = strings.TrimSpace(raw)

	if idx := strings.Index(raw, "{"); idx >= 0 {
		jsonPart := raw[idx:]
		var resp ragResponse
		if err := json.Unmarshal([]byte(jsonPart), &resp); err == nil {
			if resp.Action == "answer" || resp.Action == "retrieve" {
				return resp.Action, resp.Content, resp.Query
			}
		}
	}

	var resp ragResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil {
		if resp.Action == "answer" || resp.Action == "retrieve" {
			return resp.Action, resp.Content, resp.Query
		}
	}

	return "answer", raw, ""
}

func buildRAGUserContent(query, intermediary, referenceText, language string) string {
	var sb strings.Builder

	sb.WriteString("Original Query:\n")
	sb.WriteString(query)
	sb.WriteString("\n\n")

	if intermediary != "" {
		sb.WriteString("Current Understanding:\n")
		sb.WriteString(intermediary)
		sb.WriteString("\n\n")
	}

	if referenceText != "" {
		sb.WriteString("Reference Information:\n")
		sb.WriteString(referenceText)
		sb.WriteString("\n\n")
	}

	if language != "" {
		sb.WriteString("Please respond in: ")
		sb.WriteString(language)
	}

	return sb.String()
}

func buildReferenceText(chunks []*types.SearchResult) string {
	var sb strings.Builder
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] ", i+1))
		if chunk.KnowledgeTitle != "" {
			sb.WriteString(fmt.Sprintf("(Source: %s) ", chunk.KnowledgeTitle))
		}
		sb.WriteString(chunk.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func appendAccumulatedRefText(existing string, newChunks []*types.SearchResult) []*types.SearchResult {
	existingCount := 0
	if existing != "" {
		for _, c := range existing {
			if c == '[' {
				existingCount++
			}
		}
	}

	var all []*types.SearchResult
	_ = existingCount
	all = append(all, newChunks...)
	return all
}

func tryForceAnswer(
	ctx context.Context,
	chatModel chat.Chat,
	opt *chat.ChatOptions,
	chatManage *types.ChatManage,
	state *types.RAGIterationState,
	referenceText string,
) string {
	systemPrompt := defaultRAGForceAnswerSystemPrompt
	if chatManage.RAGRetrievalPrompt != "" {
		systemPrompt = chatManage.RAGRetrievalPrompt
	}

	userContent := buildRAGUserContent(chatManage.Query, state.Intermediary, referenceText, chatManage.Language)

	messages := []chat.Message{
		{Role: "system", Content: systemPrompt},
	}
	for _, h := range chatManage.History {
		messages = append(messages, chat.Message{Role: "user", Content: h.Query})
		messages = append(messages, chat.Message{Role: "assistant", Content: h.Answer})
	}
	messages = append(messages, chat.Message{Role: "user", Content: userContent})

	response, err := chatModel.Chat(ctx, messages, opt)
	if err != nil {
		logger.Warnf(ctx, "RAG iterate force answer failed: %v", err)
		return ""
	}

	if response == nil {
		return ""
	}

	action, content, _ := parseRAGResponse(response.Content)
	if action == "answer" && content != "" {
		return content
	}

	return response.Content
}
