<template>
  <div class="rag-iteration-display">
    <div v-if="pipelineStages && Object.keys(pipelineStages).length > 0" class="rag-timeline">
      <div
        v-for="(step, key) in sortedRAGSteps"
        :key="key"
        class="rag-step"
        :class="{ 'step-completed': step.action === 'answer', 'step-retrieving': step.action === 'retrieve' }"
      >
        <div class="step-header" @click="toggleExpand(key as string)">
          <div class="step-indicator">
            <t-icon v-if="step.action === 'answer'" name="check-circle" class="step-icon completed" />
            <t-icon v-else-if="step.action === 'retrieve'" name="refresh" class="step-icon retrieving" />
            <span v-else class="step-number">{{ extractRoundNumber(key) }}</span>
          </div>
          <div class="step-info">
            <div class="step-title">
              {{ step.action === 'answer' ? $t('agent.ragIterate.stepCompleted') : $t('agent.ragIterate.stepRetrieving') }}
              {{ extractRoundNumber(key) > 0 ? $t('agent.ragIterate.round', { round: extractRoundNumber(key) }) : '' }}
            </div>
            <div v-if="step.action === 'retrieve' && step.retrieveQuery" class="step-query">
              {{ $t('agent.ragIterate.query') }}: {{ step.retrieveQuery }}
            </div>
            <div v-if="step.action === 'retrieve' && step.chunkCount" class="step-chunks">
              {{ $t('agent.ragIterate.chunksFound', { count: step.chunkCount }) }}
            </div>
          </div>
          <div class="step-expand">
            <t-icon :name="expandedSteps[key as string] ? 'chevron-up' : 'chevron-down'" />
          </div>
        </div>
        <transition name="slide">
          <div v-if="expandedSteps[key as string] && step.content" class="step-content">
            <div class="step-content-text" v-html="renderMarkdown(step.content)"></div>
          </div>
        </transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

interface RAGIterationStep {
  action?: string;
  content?: string;
  retrieveQuery?: string;
  chunkCount?: number;
}

const props = defineProps<{
  pipelineStages?: Record<string, any>;
}>();

const expandedSteps = ref<Record<string, boolean>>({});

const sortedRAGSteps = computed(() => {
  if (!props.pipelineStages) return {};

  const steps: Record<string, RAGIterationStep> = {};
  for (const [key, value] of Object.entries(props.pipelineStages)) {
    if (key.startsWith('rag_round_') && value) {
      steps[key] = value as RAGIterationStep;
    }
  }
  return steps;
});

const extractRoundNumber = (key: string): number => {
  const match = key.match(/rag_round_(\d+)/);
  return match ? parseInt(match[1], 10) : 0;
};

const toggleExpand = (key: string) => {
  expandedSteps.value[key] = !expandedSteps.value[key];
};

const renderMarkdown = (content: string): string => {
  if (!content) return '';
  let html = content
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    .replace(/`(.*?)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>');
  return html;
};
</script>

<style scoped lang="less">
.rag-iteration-display {
  padding: 12px 0;
}

.rag-timeline {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rag-step {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color-focus);
  }

  &.step-completed {
    border-left: 3px solid var(--td-success-color);

    .step-indicator {
      background: rgba(7, 192, 95, 0.1);
      color: var(--td-success-color);
    }
  }

  &.step-retrieving {
    border-left: 3px solid #0096c7;

    .step-indicator {
      background: rgba(0, 150, 199, 0.1);
      color: #0096c7;
    }
  }
}

.step-header {
  display: flex;
  align-items: center;
  padding: 12px;
  cursor: pointer;
  user-select: none;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.step-indicator {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-right: 12px;
  background: var(--td-bg-color-secondarycontainer);

  .step-icon {
    &.completed {
      color: var(--td-success-color);
    }

    &.retrieving {
      color: #0096c7;
    }
  }

  .step-number {
    font-size: 12px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }
}

.step-info {
  flex: 1;
  min-width: 0;
}

.step-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  margin-bottom: 4px;
}

.step-query {
  font-size: 13px;
  color: var(--td-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.step-chunks {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
  margin-top: 2px;
}

.step-expand {
  flex-shrink: 0;
  margin-left: 8px;
  color: var(--td-text-color-placeholder);
}

.step-content {
  padding: 0 12px 12px 56px;
}

.step-content-text {
  font-size: 13px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 6px;
  padding: 10px 12px;

  :deep(strong) {
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  :deep(code) {
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 12px;
    background: var(--td-bg-color-container);
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--td-brand-color);
  }
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.slide-enter-to,
.slide-leave-from {
  max-height: 500px;
}
</style>