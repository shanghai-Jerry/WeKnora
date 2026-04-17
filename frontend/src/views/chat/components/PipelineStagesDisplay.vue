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
      <!-- Query Rewriting Stage -->
      <div v-if="pipelineStages.queryRewritten" class="stage-item">
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

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
    (props.pipelineStages?.expansions && props.pipelineStages.expansions.length > 0)
  );
});

const stagesCount = computed(() => {
  let count = 0;
  if (props.pipelineStages?.queryRewritten) count++;
  if (props.pipelineStages?.retrievalQuery) count++;
  if (props.pipelineStages?.vectorQuery) count++;
  if (props.pipelineStages?.keywordQuery) count++;
  if (props.pipelineStages?.expansions && props.pipelineStages.expansions.length > 0) count++;
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
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stage-item {
  padding: 10px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 8px;
  border: 1px solid var(--td-component-stroke);

  .stage-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--td-text-color-secondary);
    margin-bottom: 8px;
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
</style>
