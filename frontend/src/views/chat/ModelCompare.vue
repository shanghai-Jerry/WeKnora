<template>
  <div class="model-compare">
    <div class="compare-header">
      <div class="header-left">
        <div class="header-title">{{ t('chat.modelCompareTitle') }}</div>
        <div class="header-desc">{{ t('chat.modelCompareDesc') }}</div>
      </div>
      <div class="header-right">
        <t-button variant="outline" size="small" @click="handleClearConversation">
          <t-icon name="delete" />
          {{ t('chat.clearConversation') }}
        </t-button>
      </div>
    </div>

    <div class="compare-controls">
      <div class="control-row">
        <div class="control-item agent-selector">
          <div class="control-label">{{ t('chat.selectAgent') }}</div>
          <t-select
            v-model="selectedAgentId"
            filterable
            placeholder="选择智能体"
            size="small"
            @change="handleAgentChange"
          >
            <t-option
              v-for="agent in availableAgents"
              :key="agent.id"
              :value="agent.id"
              :label="agent.name"
            />
          </t-select>
        </div>

        <div class="control-item knowledge-selector">
          <div class="control-label">{{ t('chat.selectKnowledge') }}</div>
          <div ref="atButtonRef" class="kb-btn" :class="{ 'active': selectedItems.length > 0 }" @click.stop @mousedown.prevent="triggerMention">
            <svg width="16" height="16" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg" class="control-icon at-icon">
              <circle cx="10" cy="10" r="3.5" stroke="currentColor" stroke-width="1.8"/>
              <path d="M13.5 10V11.5C13.5 12.163 13.7634 12.7989 14.2322 13.2678C14.7011 13.7366 15.337 14 16 14C16.663 14 17.2989 13.7366 17.7678 13.2678C18.2366 12.7989 18.5 12.163 18.5 11.5V10C18.5 7.74566 17.6045 5.58365 16.0104 3.98959C14.4163 2.39553 12.2543 1.5 10 1.5C7.74566 1.5 5.58365 2.39553 3.98959 3.98959C2.39553 5.58365 1.5 7.74566 1.5 10C1.5 12.2543 2.39553 14.4163 3.98959 16.0104C5.58365 17.6045 7.74566 18.5 10 18.5H12" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span v-if="selectedItems.length > 0" class="kb-count">{{ selectedItems.length }}</span>
            <span v-else class="kb-label">{{ t('chat.selectKnowledgeBase') }}</span>
          </div>
          <div v-if="selectedItems.length > 0" class="selected-items">
            <span
              v-for="item in selectedItems"
              :key="item.id"
              class="mention-chip"
              :class="item.type === 'kb' ? (item.kbType === 'faq' ? 'mention-chip--faq' : 'mention-chip--kb') : 'mention-chip--file'"
            >
              <t-icon v-if="item.type === 'kb'" :name="item.kbType === 'faq' ? 'chat-bubble-help' : 'folder'" />
              <t-icon v-else name="file" />
              <span class="mention-chip__name">{{ item.name }}</span>
              <span class="mention-chip__remove" @click="removeItem(item)">×</span>
            </span>
          </div>
        </div>

        <div class="control-item models-selector">
          <div class="control-label">{{ t('chat.compareSelectModels') }} ({{ selectedModelIds.length }}/3)</div>
          <div class="models-row">
            <div
              v-for="modelId in selectedModelIds"
              :key="modelId"
              class="model-card"
            >
              <span class="model-name">{{ getModelName(modelId) }}</span>
              <t-icon name="close" class="remove-model-icon" @click="removeModelColumn(modelId)" />
            </div>
            <t-button
              v-if="selectedModelIds.length < 3"
              size="small"
              variant="outline"
              @click="showModelDropdown = true"
            >
              <t-icon name="add" />
              {{ t('chat.addModel') }}
            </t-button>
          </div>
          <div v-if="showModelDropdown" class="model-dropdown-wrapper" @click.stop>
            <div class="model-dropdown">
              <div class="dropdown-header">{{ t('chat.compareSelectModels') }}</div>
              <div class="model-list">
                <div
                  v-for="model in availableModels"
                  :key="model.id"
                  class="model-option"
                  :class="{
                    'selected': selectedModelIds.includes(model.id),
                    'disabled': !selectedModelIds.includes(model.id) && selectedModelIds.length >= 3
                  }"
                  @click="toggleModel(model.id)"
                >
                  <span class="model-option-name">{{ model.name }}</span>
                  <t-icon v-if="selectedModelIds.includes(model.id)" name="check" class="check-icon" />
                </div>
              </div>
              <div class="dropdown-footer">
                <t-button size="small" variant="primary" block @click="handleConfirmModel">
                  {{ t('common.confirm') }} ({{ selectedModelIds.length }})
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <MentionSelector
        :visible="showMention"
        :style="mentionStyle"
        :items="mentionItems"
        :hasMore="mentionHasMore"
        :loading="mentionLoading"
        v-model:activeIndex="mentionActiveIndex"
        @select="onMentionSelect"
        @loadMore="loadMoreMentionItems"
      />
    </Teleport>

    <div class="compare-content">
      <div class="input-area">
        <t-textarea
          ref="textareaRef"
          v-model="query"
          :placeholder="inputPlaceholder"
          :autosize="{ minRows: 2, maxRows: 4 }"
          @keydown="onKeydown"
        />
        <t-button variant="primary" @click="handleSend" :loading="loading" :disabled="!query.trim() || selectedModelIds.length === 0">
          {{ t('chat.send') }}
        </t-button>
      </div>

      <div class="results-container">
        <div class="results-grid" :style="{ gridTemplateColumns: `repeat(${selectedModelIds.length || 1}, 1fr)` }">
          <div
            v-for="modelId in selectedModelIds"
            :key="modelId"
            class="result-column"
          >
            <div class="column-header">
              <div class="column-title">
                <span class="model-badge">{{ getModelName(modelId) }}</span>
              </div>
              <div v-if="columnSessions[modelId]?.responseTime" class="column-time">
                {{ columnSessions[modelId].responseTime.toFixed(1) }}s
              </div>
            </div>

            <div ref="setColumnRef(modelId)" class="column-content" :data-model-id="modelId">
              <div v-if="!columnSessions[modelId] && !columnData[modelId]?.messages?.length" class="empty-column">
                {{ t('chat.compareEnterQuery') }}
              </div>
              
              <div v-else class="messages-container">
                <div
                  v-for="(msg, idx) in getMessagesForModel(modelId)"
                  :key="idx"
                  class="message-item"
                  :class="msg.role"
                >
                  <div v-if="msg.role === 'user'" class="message-content user-message">
                    <div class="message-label">{{ t('chat.you') }}:</div>
                    <div class="message-text">{{ msg.content }}</div>
                  </div>
                  <div v-else class="message-content assistant-message">
                    <div class="message-label">{{ getModelName(modelId) }}:</div>
                    <div v-if="msg.loading" class="loading-indicator">
                      <t-loading size="small" :text="t('chat.compareGenerating')" />
                    </div>
                    <div v-else>
                      <div class="message-text" v-html="msg.html"></div>
                      <div v-if="msg.groupedRefs?.length" class="message-refs">
                        <div class="refs-header" @click="msg.showRefs = !msg.showRefs">
                          <span>{{ t('chat.compareReferences', { count: getTotalRefCount(msg.groupedRefs) }) }}</span>
                          <t-icon :name="msg.showRefs ? 'chevron-up' : 'chevron-down'" />
                        </div>
                        <div v-if="msg.showRefs" class="refs-list">
                          <div
                            v-for="(group, groupIdx) in msg.groupedRefs"
                            :key="groupIdx"
                            class="ref-item"
                          >
                            <div class="ref-header">
                              <span class="ref-index">#{{ groupIdx + 1 }}</span>
                              <span class="ref-title">{{ group.title }}</span>
                              <span v-if="group.chunks.length > 1" class="chunk-count">{{ group.chunks.length }} 片段</span>
                            </div>
                            <div class="ref-scores">
                              <span v-if="group.maxScore != null" class="score-badge s">R:{{ formatScore(group.maxScore) }}</span>
                            </div>
                            <div class="chunk-list">
                              <t-popup
                                v-for="(chunk, chunkIdx) in group.chunks"
                                :key="chunkIdx"
                                placement="top"
                                :show-arrow="true"
                                :delay="[100, 0]"
                              >
                                <div class="chunk-preview">
                                  <span class="chunk-index">{{ chunkIdx + 1 }}</span>
                                </div>
                                <template #content>
                                  <div class="chunk-content-popup">
                                    <div class="chunk-content-header">
                                      <span class="chunk-title">{{ group.title }}</span>
                                      <span class="chunk-number">片段 {{ chunkIdx + 1 }}</span>
                                    </div>
                                    <div class="chunk-content-body" v-html="formatChunkContent(chunk.content)"></div>
                                    <div v-if="chunk.keyword_score || chunk.vector_score || chunk.rerank_score" class="chunk-scores">
                                      <span v-if="chunk.keyword_score != null" class="score-badge k">K:{{ formatScore(chunk.keyword_score) }}</span>
                                      <span v-if="chunk.vector_score != null" class="score-badge v">V:{{ formatScore(chunk.vector_score) }}</span>
                                      <span v-if="chunk.rerank_score != null" class="score-badge r">R:{{ formatScore(chunk.rerank_score) }}</span>
                                    </div>
                                  </div>
                                </template>
                              </t-popup>
                            </div>
                          </div>
                        </div>
                      </div>
                      <!-- Pipeline Stages Display -->
                      <div v-if="msg.pipelineStages && hasPipelineStages(msg.pipelineStages)" class="message-pipeline">
                        <PipelineStagesDisplay :pipeline-stages="msg.pipelineStages"></PipelineStagesDisplay>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="selectedModelIds.length === 0" class="empty-results">
            {{ t('chat.compareSelectModelHint') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed, nextTick } from 'vue';
import { useI18n } from 'vue-i18n';
import { listModels } from '@/api/model';
import { createSessions, delSession } from '@/api/chat';
import { listKnowledgeBases } from '@/api/knowledge-base';
import { listAgents, type CustomAgent } from '@/api/agent';
import { MessagePlugin } from 'tdesign-vue-next';
import { marked } from 'marked';
import { sanitizeHTML } from '@/utils/security';
import { useSettingsStore } from '@/stores/settings';
import { fetchEventSource } from '@microsoft/fetch-event-source';
import { generateRandomString } from '@/utils/index';
import i18n from '@/i18n';
import MentionSelector from '@/components/MentionSelector.vue';
import PipelineStagesDisplay from './components/PipelineStagesDisplay.vue';

const { t } = useI18n();
const settingsStore = useSettingsStore();

const query = ref('');
const loading = ref(false);
const selectedModelIds = ref<string[]>([]);
const selectedAgentId = ref('');
const showModelDropdown = ref(false);
const columnData = reactive<Record<string, any>>({});
const columnSessions = reactive<Record<string, { id: string; responseTime: number }>>({});
const columnRefs = reactive<Record<string, HTMLElement | null>>({});
const availableModels = ref<any[]>([]);
const availableAgents = ref<CustomAgent[]>([]);

const showMention = ref(false);
const mentionQuery = ref("");
const mentionItems = ref<Array<{ id: string; name: string; type: 'kb' | 'file'; kbType?: 'document' | 'faq'; count?: number }>>([]);
const mentionActiveIndex = ref(0);
const mentionStyle = ref<Record<string, string>>({});
const textareaRef = ref<any>(null);
const mentionStartPos = ref(0);
const mentionHasMore = ref(false);
const mentionLoading = ref(false);
const mentionOffset = ref(0);
const MENTION_PAGE_SIZE = 20;
const atButtonRef = ref<HTMLElement>();

const selectedItems = ref<Array<{ id: string; name: string; type: 'kb' | 'file'; kbType?: 'document' | 'faq' }>>([]);

const inputPlaceholder = computed(() => {
  const hasKnowledge = selectedItems.value.length > 0;
  return hasKnowledge ? t('input.placeholderWithContext') : t('input.placeholder');
});

onMounted(async () => {
  await loadModels();
  await loadAgents();
  await loadKnowledgeBases();
  
  selectedAgentId.value = settingsStore.selectedAgentId || '';
  
  document.addEventListener('click', handleDocumentClick);
});

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick);
});

const handleDocumentClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement;
  const modelsSelector = document.querySelector('.models-selector');
  if (modelsSelector && !modelsSelector.contains(target)) {
    showModelDropdown.value = false;
  }
};

const loadModels = async () => {
  try {
    const res = await listModels();
    availableModels.value = (res || []).filter((m: any) => m.type === 'KnowledgeQA').map((m: any) => ({
      id: m.id,
      name: m.name || m.id,
    }));
  } catch (e) {
    console.error('Failed to load models:', e);
  }
};

const loadAgents = async () => {
  try {
    const res = await listAgents();
    if (res?.data) {
      availableAgents.value = res.data;
    }
  } catch (e) {
    console.error('Failed to load agents:', e);
  }
};

const loadKnowledgeBases = async () => {
  try {
    const response: any = await listKnowledgeBases();
    if (response.data && Array.isArray(response.data)) {
      knowledgeBases.value = response.data.filter((kb: any) =>
        kb.embedding_model_id && kb.embedding_model_id !== '' &&
        kb.summary_model_id && kb.summary_model_id !== ''
      );
    }
  } catch (error) {
    console.error('Failed to load knowledge bases:', error);
  }
};

const knowledgeBases = ref<Array<{ id: string; name: string; type?: 'document' | 'faq'; knowledge_count?: number; chunk_count?: number }>>([]);

const setColumnRef = (modelId: string) => {
  return (el: any) => {
    if (el) {
      columnRefs[modelId] = el;
    }
  };
};

const getModelName = (modelId: string): string => {
  const opt = availableModels.value.find((o: any) => o.id === modelId);
  return opt?.name || modelId;
};

const formatScore = (score: number | undefined): string => {
  if (score == null) return '-';
  return score.toFixed(2);
};

interface RefChunk {
  content?: string;
  keyword_score?: number;
  vector_score?: number;
  rerank_score?: number;
  score?: number;
  chunk_index?: number;
}

interface GroupedRef {
  knowledge_base_id: string;
  knowledge_id: string;
  title: string;
  chunks: RefChunk[];
  maxScore: number | null;
}

const groupReferences = (refs: any[]): GroupedRef[] => {
  const grouped = new Map<string, GroupedRef>();
  
  for (const ref of refs) {
    const key = `${ref.knowledge_base_id || ''}-${ref.knowledge_id || ''}`;
    
    if (!grouped.has(key)) {
      grouped.set(key, {
        knowledge_base_id: ref.knowledge_base_id || '',
        knowledge_id: ref.knowledge_id || ref.knowledge_base_id || '',
        title: ref.knowledge_title || ref.knowledge_filename || `文档 ${key}`,
        chunks: [],
        maxScore: null
      });
    }
    
    const group = grouped.get(key)!;
    
    const chunk: RefChunk = {
      content: ref.content,
      keyword_score: ref.keyword_score,
      vector_score: ref.vector_score,
      rerank_score: ref.rerank_score,
      score: ref.score,
      chunk_index: ref.chunk_index
    };
    
    group.chunks.push(chunk);
    
    if (ref.rerank_score != null) {
      if (group.maxScore == null || ref.rerank_score > group.maxScore) {
        group.maxScore = ref.rerank_score;
      }
    } else if (ref.score != null) {
      if (group.maxScore == null || ref.score > group.maxScore) {
        group.maxScore = ref.score;
      }
    }
  }
  
  return Array.from(grouped.values()).sort((a, b) => {
    if (a.maxScore == null && b.maxScore == null) return 0;
    if (a.maxScore == null) return 1;
    if (b.maxScore == null) return -1;
    return b.maxScore - a.maxScore;
  });
};

const getTotalRefCount = (groupedRefs: GroupedRef[]): number => {
  return groupedRefs.reduce((sum, g) => sum + g.chunks.length, 0);
};

const formatChunkContent = (content: string | undefined): string => {
  if (!content) return '';
  const sanitized = sanitizeHTML(content);
  return marked.parse(sanitized, { async: false }) as string;
};

interface PipelineStages {
  queryRewritten?: {
    originalQuery: string;
    rewrittenQuery: string;
  };
  retrievalQuery?: string;
  vectorQuery?: string;
  keywordQuery?: string;
  expansions?: string[];
}

const hasPipelineStages = (stages: PipelineStages): boolean => {
  return (
    stages?.queryRewritten ||
    stages?.retrievalQuery ||
    stages?.vectorQuery ||
    stages?.keywordQuery ||
    (stages?.expansions && stages.expansions.length > 0)
  );
};

const toggleModel = (modelId: string) => {
  const index = selectedModelIds.value.indexOf(modelId);
  if (index !== -1) {
    selectedModelIds.value.splice(index, 1);
    delete columnData[modelId];
    delete columnSessions[modelId];
    delete columnRefs[modelId];
  } else if (selectedModelIds.value.length < 3) {
    selectedModelIds.value.push(modelId);
    columnData[modelId] = { messages: [] };
    columnSessions[modelId] = { id: '', responseTime: 0 };
  }
};

const handleConfirmModel = () => {
  showModelDropdown.value = false;
};

const removeModelColumn = async (modelId: string) => {
  const index = selectedModelIds.value.indexOf(modelId);
  if (index !== -1) {
    if (columnSessions[modelId]?.id) {
      try {
        await delSession(columnSessions[modelId].id);
      } catch (e) {
        console.error('Failed to delete session:', e);
      }
    }
    selectedModelIds.value.splice(index, 1);
    delete columnData[modelId];
    delete columnSessions[modelId];
    delete columnRefs[modelId];
  }
};

const handleAgentChange = () => {
  settingsStore.selectedAgentId = selectedAgentId.value;
};

const triggerMention = () => {
  const button = atButtonRef.value;
  if (!button) return;

  showMention.value = true;
  mentionQuery.value = "";

  const rect = button.getBoundingClientRect();
  const menuHeight = 320;

  const spaceAbove = rect.top;
  const spaceBelow = window.innerHeight - rect.bottom;

  if (spaceAbove > menuHeight || spaceAbove > spaceBelow) {
    mentionStyle.value = {
      left: `${rect.left}px`,
      bottom: `${window.innerHeight - rect.top + 8}px`,
      top: 'auto'
    };
  } else {
    mentionStyle.value = {
      left: `${rect.left}px`,
      top: `${rect.bottom + 8}px`,
      bottom: 'auto'
    };
  }

  loadMentionItems("");
};

const onMentionSelect = (item: any) => {
  if (item.type === 'kb') {
    selectedItems.value.push({
      id: item.id,
      name: item.name,
      type: 'kb',
      kbType: item.kbType || 'document'
    });
  } else if (item.type === 'file') {
    selectedItems.value.push({
      id: item.id,
      name: item.name,
      type: 'file'
    });
  }

  showMention.value = false;
};

const removeItem = (item: any) => {
  const index = selectedItems.value.findIndex(i => i.id === item.id);
  if (index !== -1) {
    selectedItems.value.splice(index, 1);
  }
};

const loadMentionItems = async (search: string, reset: boolean = false) => {
  if (reset) {
    mentionOffset.value = 0;
  }

  mentionLoading.value = true;
  try {
    const items: Array<{ id: string; name: string; type: 'kb' | 'file'; kbType?: 'document' | 'faq'; knowledge_count?: number; chunk_count?: number }> = [];

    const filteredKbs = knowledgeBases.value.filter(kb =>
      kb.name.toLowerCase().includes(search.toLowerCase()) &&
      !selectedItems.value.find(i => i.id === kb.id && i.type === 'kb')
    );

    items.push(...filteredKbs.slice(mentionOffset.value, mentionOffset.value + MENTION_PAGE_SIZE).map(kb => ({
      id: kb.id,
      name: kb.name,
      type: 'kb' as const,
      kbType: kb.type || 'document',
      knowledge_count: kb.knowledge_count,
      chunk_count: kb.chunk_count
    })));

    mentionItems.value = items;
    mentionHasMore.value = filteredKbs.length > mentionOffset.value + MENTION_PAGE_SIZE;
  } catch (error) {
    console.error('Failed to load mention items:', error);
  } finally {
    mentionLoading.value = false;
  }
};

const loadMoreMentionItems = async () => {
  if (!mentionHasMore.value || mentionLoading.value) return;
  mentionOffset.value += MENTION_PAGE_SIZE;
  await loadMentionItems(mentionQuery.value);
};

const onKeydown = (event: any) => {
  if (event.e.keyCode === 13 && !event.e.shiftKey) {
    event.e.preventDefault();
    if (query.value.trim()) {
      handleSend();
    }
  }
};

const getMessagesForModel = (modelId: string) => {
  return columnData[modelId]?.messages || [];
};

const scrollToBottom = (modelId: string) => {
  nextTick(() => {
    const column = columnRefs[modelId];
    if (column) {
      column.scrollTop = column.scrollHeight;
    }
  });
};

const getToken = (): string | null => {
  return localStorage.getItem('weknora_token');
};

const getTenantIdHeader = (): string | null => {
  const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id');
  const defaultTenantId = localStorage.getItem('weknora_tenant');
  if (selectedTenantId) {
    try {
      const defaultTenant = defaultTenantId ? JSON.parse(defaultTenantId) : null;
      const defaultId = defaultTenant?.id ? String(defaultTenant.id) : null;
      if (selectedTenantId !== defaultId) {
        return selectedTenantId;
      }
    } catch (e) {
      console.error('Failed to parse tenant info', e);
    }
  }
  return null;
};

interface StreamParams {
  sessionId: string;
  query: string;
  summaryModelId: string;
  knowledgeBaseIds: string[];
  agentEnabled: boolean;
  agentId: string;
  url: string;
}

const startModelStream = async (params: StreamParams, modelId: string, assistantMessageIndex: number) => {
  const apiUrl = import.meta.env.VITE_IS_DOCKER ? "" : "http://localhost:8080";
  const token = getToken();
  if (!token) {
    throw new Error(i18n.global.t('error.tokenNotFound'));
  }

  const url = `${apiUrl}${params.url}/${params.sessionId}`;
  const tenantIdHeader = getTenantIdHeader();

  const postBody: any = {
    query: params.query,
    agent_enabled: params.agentEnabled,
    channel: "web"
  };

  if (params.knowledgeBaseIds && params.knowledgeBaseIds.length > 0) {
    postBody.knowledge_base_ids = params.knowledgeBaseIds;
  }
  if (params.agentId) {
    postBody.agent_id = params.agentId;
  }
  if (params.summaryModelId) {
    postBody.summary_model_id = params.summaryModelId;
  }

  let fullContent = '';

  const controller = new AbortController();

  try {
    await fetchEventSource(url, {
      method: 'POST',
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${token}`,
        "Accept-Language": i18n.global.locale?.value || localStorage.getItem('locale') || 'zh-CN',
        "X-Request-ID": `${generateRandomString(12)}`,
        ...(tenantIdHeader ? { "X-Tenant-ID": tenantIdHeader } : {}),
      },
      body: JSON.stringify(postBody),
      signal: controller.signal,
      openWhenHidden: true,

      onopen: async (res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
      },

      onmessage: (ev) => {
        try {
          const data = JSON.parse(ev.data);
          
          if (data.response_type === 'answer' || data.response_type === 'complete') {
            fullContent += data.content || '';
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              columnData[modelId].messages[assistantMessageIndex].content = fullContent;
              columnData[modelId].messages[assistantMessageIndex].html = marked.parse(sanitizeHTML(fullContent)) as string;
              scrollToBottom(modelId);
            }
          } else if (data.response_type === 'references') {
            const refs = data.knowledge_references || data.data?.references || [];
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              columnData[modelId].messages[assistantMessageIndex].knowledgeRefs = refs;
              columnData[modelId].messages[assistantMessageIndex].groupedRefs = groupReferences(refs);
            }
          } else if (data.response_type === 'query_rewritten') {
            // Handle query rewritten event for pipeline stages
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              if (!columnData[modelId].messages[assistantMessageIndex].pipelineStages) {
                columnData[modelId].messages[assistantMessageIndex].pipelineStages = {};
              }
              columnData[modelId].messages[assistantMessageIndex].pipelineStages.queryRewritten = {
                originalQuery: data.data?.original_query || '',
                rewrittenQuery: data.content || data.data?.rewritten_query || ''
              };
            }
          } else if (data.response_type === 'retrieval_query') {
            // Handle retrieval query event for pipeline stages
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              if (!columnData[modelId].messages[assistantMessageIndex].pipelineStages) {
                columnData[modelId].messages[assistantMessageIndex].pipelineStages = {};
              }
              columnData[modelId].messages[assistantMessageIndex].pipelineStages.retrievalQuery = data.content || data.data?.query || '';
            }
          } else if (data.response_type === 'vector_query') {
            // Handle vector query event for pipeline stages
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              if (!columnData[modelId].messages[assistantMessageIndex].pipelineStages) {
                columnData[modelId].messages[assistantMessageIndex].pipelineStages = {};
              }
              columnData[modelId].messages[assistantMessageIndex].pipelineStages.vectorQuery = data.content || data.data?.query || '';
            }
          } else if (data.response_type === 'keyword_query') {
            // Handle keyword query event for pipeline stages
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              if (!columnData[modelId].messages[assistantMessageIndex].pipelineStages) {
                columnData[modelId].messages[assistantMessageIndex].pipelineStages = {};
              }
              columnData[modelId].messages[assistantMessageIndex].pipelineStages.keywordQuery = data.content || data.data?.query || '';
            }
          } else if (data.response_type === 'query_expansion') {
            // Handle query expansion event for pipeline stages
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              if (!columnData[modelId].messages[assistantMessageIndex].pipelineStages) {
                columnData[modelId].messages[assistantMessageIndex].pipelineStages = {};
              }
              columnData[modelId].messages[assistantMessageIndex].pipelineStages.expansions = data.data?.expansions || [];
            }
          } else if (data.response_type === 'agent_complete' || data.done) {
            if (columnData[modelId]?.messages[assistantMessageIndex]) {
              columnData[modelId].messages[assistantMessageIndex].loading = false;
              columnData[modelId].messages[assistantMessageIndex].content = fullContent;
              columnData[modelId].messages[assistantMessageIndex].html = marked.parse(sanitizeHTML(fullContent)) as string;
              scrollToBottom(modelId);
            }
          }
        } catch (e) {
          console.error('Error parsing SSE data:', e);
        }
      },

      onerror: (err) => {
        throw new Error(`${i18n.global.t('error.streamFailed')}: ${err}`);
      },

      onclose: () => {
        if (columnData[modelId]?.messages[assistantMessageIndex]) {
          columnData[modelId].messages[assistantMessageIndex].loading = false;
          columnData[modelId].messages[assistantMessageIndex].content = fullContent;
          columnData[modelId].messages[assistantMessageIndex].html = marked.parse(sanitizeHTML(fullContent)) as string;
          scrollToBottom(modelId);
        }
      },
    });
  } finally {
    controller.abort();
  }
};

const handleSend = async () => {
  if (!query.value.trim()) {
    MessagePlugin.warning(t('chat.compareQueryRequired'));
    return;
  }

  if (selectedModelIds.value.length === 0) {
    MessagePlugin.warning(t('chat.compareNoModelSelected'));
    return;
  }

  loading.value = true;
  const currentQuery = query.value;
  query.value = '';

  const agent = availableAgents.value.find(a => a.id === selectedAgentId.value);
  const agentMode = agent?.config?.agent_mode || 'quick-answer';
  const kbIds = selectedItems.value.filter(i => i.type === 'kb').map(i => i.id);

  const promises = selectedModelIds.value.map(async (modelId) => {
    if (!columnData[modelId]) {
      columnData[modelId] = { messages: [] };
    }

    columnData[modelId].messages.push({
      role: 'user',
      content: currentQuery,
    });
    scrollToBottom(modelId);

    const startTime = Date.now();
    let sessionId = columnSessions[modelId]?.id;

    const assistantMessageIndex = columnData[modelId].messages.length;
    columnData[modelId].messages.push({
      role: 'assistant',
      content: '',
      html: '',
      loading: true,
      knowledgeRefs: [],
      groupedRefs: [],
      showRefs: false,
      pipelineStages: {}
    });

    try {
      if (!sessionId) {
        const res = await createSessions({
          name: `ModelCompare-${modelId}-${Date.now()}`,
          agent_config: {
            enabled: true,
            knowledge_bases: kbIds,
            agent_id: selectedAgentId.value,
          }
        });
        sessionId = res.data.id;
        columnSessions[modelId] = { id: sessionId, responseTime: 0 };
      }

      const endpoint = agentMode === 'smart-reasoning' 
        ? '/api/v1/agent-chat'
        : '/api/v1/knowledge-chat';

      await startModelStream({
        sessionId,
        query: currentQuery,
        summaryModelId: modelId,
        knowledgeBaseIds: kbIds,
        agentEnabled: agentMode === 'smart-reasoning',
        agentId: selectedAgentId.value,
        url: endpoint,
      }, modelId, assistantMessageIndex);

      columnData[modelId].messages[assistantMessageIndex].loading = false;
      columnSessions[modelId].responseTime = (Date.now() - startTime) / 1000;
      scrollToBottom(modelId);
    } catch (e: any) {
      columnData[modelId].messages[assistantMessageIndex].content = e.message || String(e);
      columnData[modelId].messages[assistantMessageIndex].html = marked.parse(sanitizeHTML(columnData[modelId].messages[assistantMessageIndex].content)) as string;
      columnData[modelId].messages[assistantMessageIndex].loading = false;
      scrollToBottom(modelId);
    }
  });

  await Promise.all(promises);
  loading.value = false;
};

const handleClearConversation = async () => {
  for (const modelId of selectedModelIds.value) {
    if (columnSessions[modelId]?.id) {
      try {
        await delSession(columnSessions[modelId].id);
      } catch (e) {
        console.error('Failed to delete session:', e);
      }
    }
    columnSessions[modelId] = { id: '', responseTime: 0 };
    columnData[modelId] = { messages: [] };
  }
  MessagePlugin.success(t('chat.conversationCleared'));
};
</script>

<style lang="less" scoped>
.model-compare {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  gap: 12px;
  overflow: hidden;
}

.compare-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-shrink: 0;

  .header-left {
    .header-title {
      font-size: 18px;
      font-weight: 600;
      color: var(--td-text-color-primary);
    }

    .header-desc {
      font-size: 13px;
      color: var(--td-text-color-secondary);
      margin-top: 4px;
    }
  }

  .header-right {
    display: flex;
    gap: 8px;
  }
}

.compare-controls {
  flex-shrink: 0;
  padding: 12px 16px;
  background: var(--td-bg-color-container);
  border-radius: 8px;
  border: 1px solid var(--td-component-stroke);
}

.control-row {
  display: flex;
  gap: 24px;
  align-items: flex-start;
  flex-wrap: wrap;
}

.control-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.control-label {
  font-size: 13px;
  color: var(--td-text-color-secondary);
  font-weight: 500;
}

.agent-selector {
  min-width: 180px;
  :deep(.t-select) {
    width: 180px;
  }
}

.models-selector {
  position: relative;
}

.models-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.model-card {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  font-size: 12px;

  .model-name {
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .remove-model-icon {
    cursor: pointer;
    color: var(--td-text-color-placeholder);
    transition: all 0.2s;

    &:hover {
      color: var(--td-error-color);
    }
  }
}

.model-dropdown-popup {
  :deep(.t-popup__content) {
    padding: 0;
  }
}

.model-dropdown-wrapper {
  position: relative;
  z-index: 1000;
}

.model-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  box-shadow: var(--td-shadow-2);
  min-width: 200px;
  max-width: 300px;
  max-height: 320px;
  display: flex;
  flex-direction: column;
}

.dropdown-header {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  border-bottom: 1px solid var(--td-component-stroke);
}

.model-list {
  overflow-y: auto;
  flex: 1;
  max-height: 240px;
}

.model-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.2s;

  &:hover:not(.disabled) {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.selected {
    background: var(--td-brand-color-light);
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .model-option-name {
    font-size: 13px;
    color: var(--td-text-color-primary);
  }

  .check-icon {
    color: var(--td-brand-color);
  }
}

.dropdown-footer {
  padding: 10px 12px;
  border-top: 1px solid var(--td-component-stroke);
}

.knowledge-selector {
  min-width: 200px;
}

.kb-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  font-size: 12px;
  max-width: fit-content;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.active {
    background: #E6F7FF;
    color: #1890FF;
    border-color: #1890FF;
  }

  .kb-count {
    background: #1890FF;
    color: white;
    font-size: 10px;
    padding: 1px 5px;
    border-radius: 4px;
    min-width: 16px;
    text-align: center;
  }

  .kb-label {
    opacity: 0.7;
  }
}

.selected-items {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}

.mention-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: default;
  transition: background 0.2s, border-color 0.2s, box-shadow 0.2s;
  border: 0.5px solid transparent;
  color: var(--td-text-color-primary, #1f2937);
  line-height: 1.3;

  &--kb {
    background: #e6f7ff;
    border-color: #91d5ff;
    color: #0958d9;
  }

  &--faq {
    background: #fff7e6;
    border-color: #ffd591;
    color: #d46b08;
  }

  &--file {
    background: #f9f0ff;
    border-color: #d3adf7;
    color: #531dab;
  }

  &__name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__remove {
    cursor: pointer;
    opacity: 0.7;
    margin-left: 2px;

    &:hover {
      opacity: 1;
    }
  }
}

.compare-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.input-area {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  flex-shrink: 0;

  :deep(.t-textarea) {
    flex: 1;
  }

  :deep(.t-button) {
    height: 100%;
    min-height: 60px;
  }
}

.results-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.results-grid {
  display: grid;
  gap: 12px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.result-column {
  display: flex;
  flex-direction: column;
  background: var(--td-bg-color-container);
  border-radius: 8px;
  border: 1px solid var(--td-component-stroke);
  overflow: hidden;
  min-height: 0;
}

.column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border-bottom: 1px solid var(--td-component-stroke);
  flex-shrink: 0;

  .column-title {
    display: flex;
    align-items: center;
    gap: 8px;

    .model-badge {
      font-size: 13px;
      font-weight: 600;
      color: var(--td-text-color-primary);
    }
  }

  .column-time {
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

.column-content {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.messages-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-item {
  display: flex;
  flex-direction: column;
  gap: 4px;

  &.user {
    align-items: flex-end;
  }

  &.assistant {
    align-items: flex-start;
  }
}

.message-content {
  max-width: 85%;
  padding: 8px 12px;
  border-radius: 8px;
  line-height: 1.5;

  .message-label {
    font-size: 11px;
    color: var(--td-text-color-secondary);
    margin-bottom: 4px;
    font-weight: 500;
  }

  .message-text {
    font-size: 13px;
    word-wrap: break-word;

    :deep(.markdown-content) {
      word-wrap: break-word;
    }
  }
}

.user-message {
  background: var(--td-brand-color-light);
  color: var(--td-text-color-primary);
  border: 1px solid var(--td-brand-color);
}

.assistant-message {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  border: 1px solid var(--td-component-stroke);
}

.loading-indicator {
  display: flex;
  align-items: center;
  min-height: 40px;
}

.message-refs {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--td-component-stroke);
}

.message-pipeline {
  margin-top: 8px;
}

.refs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;

  &:hover {
    background: var(--td-bg-color-container);
  }
}

.refs-list {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ref-item {
  padding: 6px 8px;
  border-radius: 4px;
  background: var(--td-bg-color-container);
  font-size: 11px;
}

.ref-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.ref-index {
  font-weight: 600;
  color: var(--td-text-color-placeholder);
  flex-shrink: 0;
}

.ref-title {
  flex: 1;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ref-scores {
  display: flex;
  gap: 3px;
  flex-wrap: wrap;
}

.score-badge {
  font-size: 9px;
  padding: 1px 3px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', monospace;

  &.k {
    background: #e6f7ff;
    color: #1890ff;
  }

  &.v {
    background: #f6ffed;
    color: #52c41a;
  }

  &.r {
    background: #fff7e6;
    color: #fa8c16;
  }

  &.s {
    background: #f9f0ff;
    color: #722ed1;
  }
}

.chunk-count {
  font-size: 10px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  padding: 1px 4px;
  border-radius: 3px;
  flex-shrink: 0;
}

.chunk-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.chunk-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: var(--td-brand-color-light);
    border-color: var(--td-brand-color);
  }

  .chunk-index {
    font-size: 10px;
    font-weight: 500;
    color: var(--td-text-color-secondary);
  }
}

.chunk-content-popup {
  max-width: 400px;
  max-height: 300px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.chunk-content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--td-component-stroke);
  margin-bottom: 8px;

  .chunk-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 280px;
  }

  .chunk-number {
    font-size: 11px;
    color: var(--td-text-color-secondary);
    flex-shrink: 0;
  }
}

.chunk-content-body {
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-primary);
  max-height: 200px;
  overflow-y: auto;
  word-break: break-word;

  :deep(p) {
    margin: 0 0 8px 0;
  }

  :deep(p:last-child) {
    margin-bottom: 0;
  }
}

.chunk-scores {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--td-component-stroke);
}

.empty-column {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
}

.empty-results {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--td-text-color-placeholder);
  text-align: center;
  font-size: 13px;
  background: var(--td-bg-color-container);
  border-radius: 8px;
  border: 1px solid var(--td-component-stroke);
}

@media (max-width: 900px) {
  .control-row {
    flex-direction: column;
  }

  .agent-selector,
  .knowledge-selector,
  .models-selector {
    width: 100%;
  }

  .results-grid {
    grid-template-columns: 1fr !important;
  }
}
</style>
