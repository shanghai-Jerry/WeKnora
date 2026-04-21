<template>
  <div v-if="hasStages" class="pipeline-stages">
    <div class="stages-header" @click="toggleExpanded">
      <div class="stages-title">
        <t-icon name="app" class="stages-icon" />
        <span>{{ $t('chat.pipelineStages') }}</span>
        <span class="stages-count">{{ stagesCount }}</span>
      </div>
      <t-icon :name="expanded ? 'chevron-up' : 'chevron-down'" class="toggle-icon" />
    </div>

    <div v-show="expanded" class="stages-content">
      <div class="timeline-track"></div>

      <!-- Query Rewriting Stage -->
      <div v-if="pipelineStages.queryRewritten" class="stage-item">
        <div class="timeline-dot"></div>
        <div class="stage-label">
          <t-icon name="edit-1" class="stage-icon" />
          {{ $t('chat.queryRewritten') }}
        </div>
        <div class="stage-content">
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
        <div class="stage-content">
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
        <div class="stage-content">
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
        <div class="stage-content">
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
        <div class="stage-content">
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

      <!-- Intent Explore Stage (Multi-Path Retrieval) -->
      <div v-if="pipelineStages.intentExplore && pipelineStages.intentExplore.analysisPaths && pipelineStages.intentExplore.analysisPaths.length > 0" class="stage-item intent-explore-stage">
        <div class="timeline-dot"></div>
        <div class="stage-label">
          <t-icon name="node-tree" class="stage-icon" />
          {{ $t('chat.intentExplore') || '已理解问题并定位研究方向' }}
        </div>
        <div class="stage-content">
          <div class="intent-explore-info">
            <div class="intent-query-count">
              <span class="count-label">检索路径:</span>
              <span class="count-value">{{ pipelineStages.intentExplore.analysisPaths.length }} 条</span>
              <span class="count-label ml-4">召回文献:</span>
              <span class="count-value">{{ pipelineStages.intentExplore.totalSearchCount }} 篇</span>
            </div>
          </div>

          <!-- Intent Graph Visualization -->
          <div class="intent-graph-wrapper">
            <div
              v-for="path in pipelineStages.intentExplore.analysisPaths"
              :key="path.path_id"
              class="path-visual-card"
            >
              <div class="visual-graph">
                <div class="visual-center">{{ path.entity }}</div>
                <template v-for="(dim, idx) in path.dimensions.slice(0, 4)" :key="dim">
                  <div class="visual-dim" :class="`pos-${idx}`">{{ dim }}</div>
                  <div class="visual-line" :class="`line-${idx}`"></div>
                </template>
              </div>
              <div v-if="path.merged_search_string" class="visual-search-string">
                {{ path.merged_search_string }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

interface AnalysisPath {
  path_id: number;
  entity: string;
  dimensions: string[];
  merged_search_string: string;
  reason: string;
}

interface IntentExploreData {
  originalQuery: string;
  analysisPaths: AnalysisPath[];
  finalSearchQueries: string[];
  totalSearchCount: number;
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
}>();

const expanded = ref(true);

const hasStages = computed(() => {
  return (
    props.pipelineStages?.queryRewritten ||
    props.pipelineStages?.retrievalQuery ||
    props.pipelineStages?.vectorQuery ||
    props.pipelineStages?.keywordQuery ||
    (props.pipelineStages?.expansions && props.pipelineStages.expansions.length > 0) ||
    (props.pipelineStages?.intentExplore && props.pipelineStages.intentExplore.analysisPaths && props.pipelineStages.intentExplore.analysisPaths.length > 0)
  );
});

const stagesCount = computed(() => {
  let count = 0;
  if (props.pipelineStages?.queryRewritten) count++;
  if (props.pipelineStages?.retrievalQuery) count++;
  if (props.pipelineStages?.vectorQuery) count++;
  if (props.pipelineStages?.keywordQuery) count++;
  if (props.pipelineStages?.expansions && props.pipelineStages.expansions.length > 0) count++;
  if (props.pipelineStages?.intentExplore && props.pipelineStages.intentExplore.analysisPaths && props.pipelineStages.intentExplore.analysisPaths.length > 0) count++;
  return count;
});

const toggleExpanded = () => {
  expanded.value = !expanded.value;
};
</script>

<style lang="less" scoped>
.pipeline-stages {
  margin-top: 16px;
  border: 1px solid var(--td-brand-color-focus);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stages-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: linear-gradient(135deg, var(--td-brand-color-light) 0%, var(--td-bg-color-secondarycontainer) 100%);
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.stages-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--td-brand-color);

  .stages-icon {
    font-size: 16px;
  }
}

.stages-count {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 6px;
  background: var(--td-brand-color);
  color: white;
  border-radius: 10px;
}

.toggle-icon {
  font-size: 14px;
  color: var(--td-text-color-secondary);
}

.stages-content {
  position: relative;
  padding: 16px 14px 16px 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.timeline-track {
  position: absolute;
  left: 18px;
  top: 24px;
  bottom: 24px;
  width: 2px;
  background: var(--td-component-stroke);
  border-radius: 1px;
}

.stage-item {
  position: relative;
  padding: 12px 14px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 10px;
  border: 1px solid var(--td-component-stroke);
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color-focus);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }

  .timeline-dot {
    position: absolute;
    left: -19px;
    top: 18px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--td-brand-color);
    border: 2px solid var(--td-bg-color-container);
    box-shadow: 0 0 0 2px var(--td-brand-color-light);
    z-index: 2;
  }

  .stage-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--td-text-color-secondary);
    margin-bottom: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;

    .stage-icon {
      font-size: 12px;
      color: var(--td-brand-color);

      &.vector-icon {
        color: #722ed1;
      }

      &.keyword-icon {
        color: #1890ff;
      }
    }
  }
}

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

/* Intent Explore Graph Styles */
.intent-explore-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.intent-query-count {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  padding: 6px 10px;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.08);

  .count-label {
    color: var(--td-text-color-secondary);
  }

  .count-value {
    color: var(--td-brand-color);
    font-weight: 500;
  }

  .ml-4 {
    margin-left: 16px;
  }
}

.intent-graph-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  justify-content: center;
  margin-top: 12px;
}

.path-visual-card {
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
  max-width: 260px;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color-focus);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  }
}

.visual-graph {
  position: relative;
  width: 200px;
  height: 150px;
  margin: 0 auto;
}

.visual-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #f0fffe;
  border: 2px solid #5cdbd3;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #006d75;
  z-index: 2;
  text-align: center;
  padding: 4px;
  word-break: break-word;
  line-height: 1.2;
}

.visual-dim {
  position: absolute;
  padding: 4px 10px;
  border-radius: 50%;
  background: #fff;
  border: 1px solid #d9d9d9;
  font-size: 11px;
  color: #595959;
  z-index: 2;
  white-space: nowrap;
}

.visual-dim.pos-0 {
  top: 2px;
  left: 50%;
  transform: translateX(-50%);
}

.visual-dim.pos-1 {
  top: 50%;
  left: 2px;
  transform: translateY(-50%);
}

.visual-dim.pos-2 {
  top: 50%;
  right: 2px;
  transform: translateY(-50%);
}

.visual-dim.pos-3 {
  bottom: 2px;
  left: 50%;
  transform: translateX(-50%);
}

.visual-line {
  position: absolute;
  background: #d9d9d9;
  z-index: 1;
}

/* Top line: from top dim bottom to center top */
.visual-line.line-0 {
  top: 26px;
  left: 50%;
  width: 1px;
  bottom: calc(50% + 32px);
}

/* Left line: from left dim right to center left */
.visual-line.line-1 {
  top: 50%;
  left: 26px;
  height: 1px;
  right: calc(50% + 32px);
}

/* Right line: from center right to right dim left */
.visual-line.line-2 {
  top: 50%;
  left: calc(50% + 32px);
  height: 1px;
  right: 26px;
}

/* Bottom line: from center bottom to bottom dim top */
.visual-line.line-3 {
  top: calc(50% + 32px);
  left: 50%;
  width: 1px;
  bottom: 26px;
}

.visual-search-string {
  font-size: 11px;
  color: var(--td-text-color-secondary);
  line-height: 1.4;
  padding: 6px 10px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 6px;
  word-break: break-word;
  text-align: center;
  width: 100%;
}
</style>
