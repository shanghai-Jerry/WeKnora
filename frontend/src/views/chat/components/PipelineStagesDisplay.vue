<template>
  <div v-if="hasStages" class="pipeline-stages">
    <!-- Header: 循证检索折叠面板 -->
    <div class="stages-header" @click="toggleExpanded">
      <div class="stages-title">
        <svg class="header-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" fill="currentColor"/>
        </svg>
        <span>{{ $t('chat.evidenceRetrieval') || '已完成循证检索' }}</span>
      </div>
      <t-icon :name="expanded ? 'chevron-up' : 'chevron-down'" class="toggle-icon" />
    </div>

    <!-- Content: 时间轴 -->
    <div v-show="expanded" class="stages-content">
      <div class="timeline-track"></div>

      <!-- === Step 1: 已理解问题并定位研究方向 === -->
      <div v-if="hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot completed">
          <svg viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
        </div>
        <div class="stage-label">
          已理解问题并定位研究方向
        </div>
        <div class="stage-body">
          <!-- 统计信息 -->
          <div class="evidence-stats">
            <span class="stat-item">
              <span class="stat-label">检索路径</span>
              <span class="stat-value">{{ intentExplore.analysisPaths.length }} 条</span>
            </span>
            <span class="stat-divider"></span>
            <span class="stat-item">
              <span class="stat-label">召回文献</span>
              <span class="stat-value">{{ intentExplore.totalSearchCount }} 篇</span>
            </span>
          </div>

          <!-- 知识图谱卡片网格 -->
          <div class="knowledge-graph-grid">
            <div
              v-for="path in intentExplore.analysisPaths"
              :key="path.path_id"
              class="graph-card"
            >
              <!-- === 概念型路径：中心节点 + 维度 === -->
              <div v-if="isConceptPath(path)" class="graph-canvas">
                <svg class="graph-svg" viewBox="0 0 220 160" xmlns="http://www.w3.org/2000/svg">
                  <template v-for="(dim, idx) in getVisibleDimensions(path.dimensions)" :key="`line-${idx}`">
                    <line
                      :x1="getCenterPos().x"
                      :y1="getCenterPos().y"
                      :x2="getDimPos(idx, getVisibleDimensions(path.dimensions).length).x"
                      :y2="getDimPos(idx, getVisibleDimensions(path.dimensions).length).y"
                      stroke="#E5E7EB"
                      stroke-width="1"
                      stroke-dasharray="4"
                    />
                  </template>
                </svg>

                <div class="node center-node">
                  <span class="node-text">{{ path.entity }}</span>
                </div>

                <div
                  v-for="(dim, idx) in getVisibleDimensions(path.dimensions)"
                  :key="`dim-${idx}`"
                  class="node dim-node"
                  :style="getDimNodeStyle(idx, getVisibleDimensions(path.dimensions).length)"
                >
                  <span class="node-text">{{ dim }}</span>
                </div>
              </div>

              <!-- === 关系型路径：source → target === -->
              <div v-else-if="isRelationPath(path)" class="graph-canvas relation-canvas">
                <svg class="graph-svg" viewBox="0 0 220 160" xmlns="http://www.w3.org/2000/svg">
                  <line x1="45" y1="80" x2="175" y2="80" stroke="#5cdbd3" stroke-width="1.5" stroke-dasharray="4" />
                  <polygon points="170,75 180,80 170,85" fill="#5cdbd3" />
                </svg>

                <div class="node relation-node source-node">
                  <span class="node-text">{{ path.source_entity }}</span>
                </div>

                <div class="node relation-label">
                  <span class="label-text">{{ path.interaction_type || '关联' }}</span>
                </div>

                <div class="node relation-node target-node">
                  <span class="node-text">{{ path.target_entity }}</span>
                </div>
              </div>

              <!-- === 兜底：显示搜索词 === -->
              <div v-else class="graph-canvas fallback-canvas">
                <div class="fallback-query">
                  <span class="fallback-label">检索方向</span>
                  <span class="fallback-text">{{ path.merged_search_string }}</span>
                </div>
              </div>

              <!-- 路径说明（reason / clinical_significance） -->
              <div v-if="path.reason || path.clinical_significance" class="path-reason">
                {{ path.reason || path.clinical_significance }}
              </div>

              <!-- 搜索词描述 -->
              <div v-if="path.merged_search_string" class="path-description">
                {{ path.merged_search_string }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- === Step 2: 已检索X篇权威内容 === -->
      <div v-if="hasReferences || hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot" :class="{ completed: hasReferences }">
          <svg v-if="hasReferences" viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
          <span v-else class="dot-inner"></span>
        </div>
        <div class="stage-label">
          已检索{{ totalReferencesCount }}篇权威内容
          <t-icon v-if="hasReferences" name="chevron-up" class="collapse-icon" />
        </div>
        <div class="stage-body">
          <!-- 来源标识 -->
          <div class="source-header">
            <div class="source-avatars">
              <span
                v-for="(source, idx) in referenceSources.slice(0, 4)"
                :key="idx"
                class="source-avatar"
                :style="{ background: sourceColors[idx % sourceColors.length] }"
              >
                {{ source.charAt(0).toUpperCase() }}
              </span>
            </div>
            <span class="source-text">来自知识库文献等内容</span>
          </div>

          <!-- 引用卡片列表 -->
          <div v-if="hasReferences" class="citation-list">
            <div
              v-for="(ref, idx) in visibleReferences"
              :key="idx"
              class="citation-item"
            >
              <p class="citation-title">{{ ref.knowledge_title || ref.knowledge_filename || '未命名文献' }}</p>
              <p class="citation-source">{{ formatSource(ref) }}</p>
              <p v-if="ref.content" class="citation-snippet">{{ truncateContent(ref.content) }}</p>
              <div v-if="idx < visibleReferences.length - 1" class="citation-divider"></div>
            </div>
            <div v-if="knowledgeReferences.length > 5" class="citation-more">
              还有 {{ knowledgeReferences.length - 5 }} 篇文献...
            </div>
          </div>
          <div v-else class="citation-loading">
            <div class="loading-dots">
              <span></span><span></span><span></span>
            </div>
            <span class="loading-text">正在检索文献...</span>
          </div>
        </div>
      </div>

      <!-- === Step 3: 已完成引用并总结 === -->
      <div v-if="hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot completed">
          <svg viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
        </div>
        <div class="stage-label">已完成引用并总结</div>
      </div>

      <!-- === Legacy Pipeline Stages (非循证检索模式) === -->
      <template v-if="!hasIntentExplore">
        <!-- Query Rewriting Stage -->
        <div v-if="pipelineStages.queryRewritten" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="edit-1" class="stage-icon" />
            {{ $t('chat.queryRewritten') }}
          </div>
          <div class="stage-body">
            <div class="query-compare">
              <div class="query-item original">
                <span class="query-tag">{{ $t('chat.originalQuery') }}</span>
                <span class="query-text">{{ pipelineStages.queryRewritten.originalQuery }}</span>
              </div>
              <t-icon name="arrow-right" class="arrow-icon" />
              <div class="query-item rewritten">
                <span class="query-tag">{{ $t('chat.rewrittenQuery') }}</span>
                <span class="query-text">{{ pipelineStages.queryRewritten.rewrittenQuery }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Vector Retrieval Query Stage -->
        <div v-if="pipelineStages.vectorQuery" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="bar-chart" class="stage-icon vector-icon" />
            {{ $t('chat.vectorRetrieval') }}
          </div>
          <div class="stage-body">
            <div class="retrieval-query vector-query">
              <span class="query-text">{{ pipelineStages.vectorQuery }}</span>
            </div>
          </div>
        </div>

        <!-- Keyword Retrieval Query Stage -->
        <div v-if="pipelineStages.keywordQuery" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="search" class="stage-icon keyword-icon" />
            {{ $t('chat.keywordRetrieval') }}
          </div>
          <div class="stage-body">
            <div class="retrieval-query keyword-query">
              <span class="query-text">{{ pipelineStages.keywordQuery }}</span>
            </div>
          </div>
        </div>

        <!-- Unified Retrieval Query (fallback) -->
        <div v-if="pipelineStages.retrievalQuery && !pipelineStages.vectorQuery && !pipelineStages.keywordQuery" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="search" class="stage-icon" />
            {{ $t('chat.retrievalQuery') }}
          </div>
          <div class="stage-body">
            <div class="retrieval-query">
              <span class="query-text">{{ pipelineStages.retrievalQuery }}</span>
            </div>
          </div>
        </div>

        <!-- Query Expansion Stage -->
        <div v-if="pipelineStages.expansions && pipelineStages.expansions.length > 0" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="layers" class="stage-icon" />
            {{ $t('chat.queryExpansion') }}
          </div>
          <div class="stage-body">
            <div class="expansion-list">
              <span
                v-for="(expansion, idx) in pipelineStages.expansions"
                :key="idx"
                class="expansion-tag"
              >
                {{ expansion }}
              </span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

interface AnalysisPath {
  path_id: number;
  // Concept-type path
  entity?: string;
  dimensions?: string[] | null;
  reason?: string;
  // Relation-type path
  source_entity?: string;
  target_entity?: string;
  interaction_type?: string;
  mechanistic_link?: string;
  clinical_significance?: string;
  // Common
  merged_search_string: string;
}

interface IntentExploreData {
  originalQuery: string;
  analysisPaths: AnalysisPath[];
  finalSearchQueries: string[];
  totalSearchCount: number;
}

interface ReferenceItem {
  id?: string;
  knowledge_title?: string;
  knowledge_filename?: string;
  knowledge_source?: string;
  content?: string;
  score?: number;
  match_type?: string;
}

interface PipelineStages {
  queryRewritten?: {
    originalQuery: string;
    rewrittenQuery: string;
  };
  retrievalQuery?: string;
  vectorQuery?: string;
  keywordQuery?: string;
  expansions?: string[];
  intentExplore?: IntentExploreData;
}

const props = defineProps<{
  pipelineStages: PipelineStages;
  knowledgeReferences?: ReferenceItem[];
}>();

const expanded = ref(true);

const hasIntentExplore = computed(() => {
  return props.pipelineStages?.intentExplore &&
    props.pipelineStages.intentExplore.analysisPaths &&
    props.pipelineStages.intentExplore.analysisPaths.length > 0;
});

const intentExplore = computed(() => props.pipelineStages?.intentExplore);

const hasReferences = computed(() => {
  return props.knowledgeReferences && props.knowledgeReferences.length > 0;
});

const knowledgeReferences = computed(() => props.knowledgeReferences || []);

const totalReferencesCount = computed(() => {
  if (hasReferences.value) return props.knowledgeReferences!.length;
  return props.pipelineStages?.intentExplore?.totalSearchCount || 0;
});

const visibleReferences = computed(() => {
  return knowledgeReferences.value.slice(0, 5);
});

const referenceSources = computed(() => {
  const sources = new Set<string>();
  knowledgeReferences.value.forEach((ref) => {
    const source = ref.knowledge_source || ref.knowledge_filename || '未知来源';
    sources.add(source);
  });
  return Array.from(sources);
});

const sourceColors = ['#3B82F6', '#EF4444', '#F59E0B', '#10B981', '#8B5CF6', '#EC4899'];

const hasStages = computed(() => {
  return (
    props.pipelineStages?.queryRewritten ||
    props.pipelineStages?.retrievalQuery ||
    props.pipelineStages?.vectorQuery ||
    props.pipelineStages?.keywordQuery ||
    (props.pipelineStages?.expansions && props.pipelineStages.expansions.length > 0) ||
    hasIntentExplore.value
  );
});

const toggleExpanded = () => {
  expanded.value = !expanded.value;
};

// 判断路径类型
const isConceptPath = (path: AnalysisPath) => {
  return !!path.entity && !!path.dimensions && path.dimensions.length > 0;
};

const isRelationPath = (path: AnalysisPath) => {
  return !!path.source_entity && !!path.target_entity;
};

// 最多展示4个维度
const getVisibleDimensions = (dimensions: string[] | null | undefined) => {
  if (!dimensions || !Array.isArray(dimensions)) return [];
  return dimensions.slice(0, 4);
};

// 中心节点位置（相对于 graph-canvas）
const getCenterPos = () => ({ x: 110, y: 80 });

// 根据维度数量和索引计算位置（圆形分布）
const getDimPos = (index: number, total: number) => {
  const centerX = 110;
  const centerY = 80;
  const radius = 55;

  // 根据数量分布在不同角度
  let angle: number;
  if (total === 1) {
    angle = -Math.PI / 2; // 顶部
  } else if (total === 2) {
    angle = index === 0 ? -Math.PI / 2 : Math.PI / 2; // 上、下
  } else if (total === 3) {
    const angles = [-Math.PI / 2, Math.PI * 0.85, Math.PI * 0.15];
    angle = angles[index];
  } else {
    const angles = [-Math.PI / 2, Math.PI, 0, Math.PI / 2];
    angle = angles[index];
  }

  return {
    x: centerX + radius * Math.cos(angle),
    y: centerY + radius * Math.sin(angle)
  };
};

// 获取维度节点的 CSS 样式（用于定位 HTML 节点）
const getDimNodeStyle = (index: number, total: number) => {
  const pos = getDimPos(index, total);
  const canvasWidth = 220;
  const canvasHeight = 160;
  return {
    left: `${(pos.x / canvasWidth) * 100}%`,
    top: `${(pos.y / canvasHeight) * 100}%`,
    transform: 'translate(-50%, -50%)'
  };
};

const getRelevanceLabel = (score: number) => {
  if (score >= 0.85) return '高相关';
  if (score >= 0.6) return '中相关';
  if (score >= 0.35) return '低相关';
  return '弱相关';
};

const formatSource = (ref: ReferenceItem) => {
  const parts: string[] = [];
  if (ref.knowledge_filename) parts.push(ref.knowledge_filename);
  if (typeof ref.score === 'number') {
    parts.push(`相关度: ${getRelevanceLabel(ref.score)}`);
  }
  return parts.join(' · ') || '未知来源';
};

const truncateContent = (content: string) => {
  if (!content) return '';
  const maxLen = 80;
  return content.length > maxLen ? content.substring(0, maxLen) + '...' : content;
};
</script>

<style lang="less" scoped>
.pipeline-stages {
  margin-top: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.stages-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: linear-gradient(135deg, #f0fdf4 0%, var(--td-bg-color-secondarycontainer) 100%);
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
  border-bottom: 1px solid var(--td-component-stroke);

  &:hover {
    background: linear-gradient(135deg, #dcfce7 0%, var(--td-bg-color-container-hover) 100%);
  }
}

.stages-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #166534;

  .header-icon {
    width: 18px;
    height: 18px;
    color: #16a34a;
  }
}

.toggle-icon {
  font-size: 14px;
  color: var(--td-text-color-secondary);
}

.stages-content {
  position: relative;
  padding: 20px 14px 20px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.timeline-track {
  position: absolute;
  left: 24px;
  top: 28px;
  bottom: 28px;
  width: 2px;
  background: var(--td-component-stroke);
  border-radius: 1px;
}

.stage-item {
  position: relative;

  .timeline-dot {
    position: absolute;
    left: -24px;
    top: 2px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: white;
    border: 2px solid var(--td-component-stroke);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2;
    transition: all 0.3s ease;

    &.completed {
      border-color: #22c55e;
      background: #22c55e;

      .check-icon {
        width: 14px;
        height: 14px;
        color: white;
      }
    }

    .dot-inner {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--td-component-stroke);
    }
  }

  .stage-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 12px;

    .stage-icon {
      font-size: 14px;
      color: var(--td-brand-color);

      &.vector-icon {
        color: #722ed1;
      }

      &.keyword-icon {
        color: #1890ff;
      }
    }

    .collapse-icon {
      margin-left: auto;
      font-size: 12px;
      color: var(--td-text-color-secondary);
    }
  }
}

/* ===== Evidence Step Styles ===== */
.evidence-step {
  .stage-body {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 10px;
    border: 1px solid var(--td-component-stroke);
    padding: 14px;
  }
}

.evidence-stats {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);

  .stat-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .stat-label {
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }

  .stat-value {
    font-size: 13px;
    font-weight: 600;
    color: var(--td-brand-color);
  }

  .stat-divider {
    width: 1px;
    height: 14px;
    background: var(--td-component-stroke);
  }
}

/* ===== Knowledge Graph Grid ===== */
.knowledge-graph-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: flex-start;
}

.graph-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: white;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  min-width: 200px;
  flex: 1;
  max-width: 240px;
  transition: all 0.2s ease;

  &:hover {
    border-color: #5cdbd3;
    box-shadow: 0 2px 12px rgba(92, 219, 211, 0.15);
    transform: translateY(-2px);
  }
}

.graph-canvas {
  position: relative;
  width: 100%;
  height: 160px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.graph-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.node {
  position: absolute;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  text-align: center;
  z-index: 2;
  transition: all 0.3s ease;

  .node-text {
    font-size: 11px;
    line-height: 1.2;
    word-break: break-word;
    padding: 2px;
  }
}

.center-node {
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 64px;
  height: 64px;
  background: #f0fffe;
  border: 2px solid #5cdbd3;
  color: #006d75;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(92, 219, 211, 0.2);
  animation: float 3s ease-in-out infinite;

  .node-text {
    font-size: 12px;
    font-weight: 600;
  }
}

.dim-node {
  padding: 6px 10px;
  min-width: 48px;
  min-height: 36px;
  background: white;
  border: 1px solid #e5e7eb;
  color: #4b5563;
  font-size: 11px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  white-space: nowrap;

  &:hover {
    border-color: #5cdbd3;
    background: #f0fffe;
  }

  .node-text {
    font-size: 10px;
  }
}

.path-description {
  font-size: 11px;
  color: var(--td-text-color-secondary);
  line-height: 1.4;
  padding: 6px 8px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 6px;
  word-break: break-word;
  text-align: center;
  width: 100%;
}

.path-reason {
  font-size: 11px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
  padding: 6px 8px;
  background: rgba(92, 219, 211, 0.06);
  border-radius: 6px;
  border-left: 2px solid #5cdbd3;
  word-break: break-word;
  width: 100%;
}

/* ===== Relation Path Styles ===== */
.relation-canvas {
  .relation-node {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 600;
    word-break: break-word;
    padding: 4px;
    z-index: 2;
    animation: float 3s ease-in-out infinite;
  }

  .source-node {
    left: 15px;
    top: 50%;
    transform: translate(0, -50%);
    background: #f0fffe;
    border: 2px solid #5cdbd3;
    color: #006d75;
    box-shadow: 0 2px 8px rgba(92, 219, 211, 0.2);
  }

  .target-node {
    right: 15px;
    top: 50%;
    transform: translate(0, -50%);
    background: #fff7ed;
    border: 2px solid #fbbf24;
    color: #92400e;
    box-shadow: 0 2px 8px rgba(251, 191, 36, 0.2);
    animation-delay: 0.5s;
  }

  .relation-label {
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    background: white;
    border: 1px solid #d1d5db;
    color: #374151;
    font-size: 10px;
    padding: 3px 8px;
    border-radius: 12px;
    white-space: nowrap;
    z-index: 3;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);

    .label-text {
      font-size: 10px;
      font-weight: 500;
    }
  }
}

/* ===== Fallback Path Styles ===== */
.fallback-canvas {
  display: flex;
  align-items: center;
  justify-content: center;

  .fallback-query {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 12px;
    background: #f9fafb;
    border: 1px dashed #d1d5db;
    border-radius: 8px;
    text-align: center;

    .fallback-label {
      font-size: 10px;
      color: var(--td-text-color-placeholder);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .fallback-text {
      font-size: 12px;
      color: var(--td-text-color-primary);
      line-height: 1.4;
      word-break: break-word;
    }
  }
}

@keyframes float {
  0%, 100% {
    transform: translate(-50%, -50%) translateY(0);
  }
  50% {
    transform: translate(-50%, -50%) translateY(-4px);
  }
}

/* ===== Citation Styles ===== */
.source-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  margin-bottom: 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

.source-avatars {
  display: flex;
  align-items: center;

  .source-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 10px;
    font-weight: 600;
    border: 2px solid white;
    margin-left: -6px;

    &:first-child {
      margin-left: 0;
    }
  }
}

.source-text {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.citation-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.citation-item {
  padding: 10px 0;
  position: relative;
}

.citation-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.5;
  margin-bottom: 4px;
}

.citation-source {
  font-size: 11px;
  color: var(--td-text-color-secondary);
  margin-bottom: 4px;
}

.citation-snippet {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
  line-height: 1.4;
}

.citation-divider {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 1px;
  background: var(--td-component-stroke);
}

.citation-more {
  text-align: center;
  padding: 8px 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.citation-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px 0;

  .loading-text {
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

.loading-dots {
  display: flex;
  align-items: center;
  gap: 4px;

  span {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--td-brand-color);
    animation: typingBounce 1.4s ease-in-out infinite;

    &:nth-child(1) {
      animation-delay: 0s;
    }

    &:nth-child(2) {
      animation-delay: 0.2s;
    }

    &:nth-child(3) {
      animation-delay: 0.4s;
    }
  }
}

@keyframes typingBounce {
  0%, 60%, 100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-6px);
  }
}

/* ===== Legacy Styles ===== */
.query-compare {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  flex-wrap: wrap;
}

.query-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 12px;
  flex: 1;
  min-width: 120px;
  max-width: calc(50% - 20px);

  &.original {
    background: rgba(0, 0, 0, 0.04);
    border: 1px solid rgba(0, 0, 0, 0.08);
  }

  &.rewritten {
    background: var(--td-brand-color-light);
    border: 1px solid var(--td-brand-color-focus);
  }

  .query-tag {
    font-size: 10px;
    font-weight: 600;
    color: var(--td-text-color-placeholder);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .query-text {
    color: var(--td-text-color-primary);
    line-height: 1.4;
    word-break: break-word;
  }
}

.arrow-icon {
  font-size: 14px;
  color: var(--td-brand-color);
  flex-shrink: 0;
  margin-top: 8px;
}

.retrieval-query {
  padding: 8px 10px;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 6px;
  font-size: 12px;
  color: var(--td-text-color-primary);
  line-height: 1.4;
  word-break: break-word;
  border: 1px solid rgba(0, 0, 0, 0.08);

  &.vector-query {
    background: #f9f0ff;
    border-color: #d3adf7;
  }

  &.keyword-query {
    background: #e6f7ff;
    border-color: #91d5ff;
  }
}

.expansion-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.expansion-tag {
  padding: 4px 10px;
  background: white;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  font-size: 11px;
  color: var(--td-text-color-primary);
  transition: all 0.2s;

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}
</style>
