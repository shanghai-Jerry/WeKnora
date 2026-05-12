<template>
  <div v-if="fileUrl" class="html-file-card">
    <div class="file-info">
      <t-icon name="file" class="file-icon" />
      <div class="file-meta">
        <div class="file-title">{{ data.title || 'HTML Report' }}</div>
        <div class="file-name">{{ data.file_path }}</div>
      </div>
    </div>
    <div class="file-actions">
      <t-button size="small" variant="outline" @click="openFile">
        <t-icon name="jump" />
        {{ $t('agentStream.htmlReport.open') }}
      </t-button>
      <t-button size="small" variant="outline" @click="downloadFile">
        <t-icon name="download" />
        {{ $t('agentStream.htmlReport.download') }}
      </t-button>
    </div>
  </div>
  <div v-else class="html-report-container">
    <iframe
      ref="iframeRef"
      :srcdoc="htmlContent"
      class="html-report-iframe"
      sandbox="allow-scripts"
      @load="onIframeLoad"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { getDown } from '@/utils/request';
import type { HtmlReportData } from '@/types/tool-results';

interface Props {
  data: HtmlReportData;
  output?: string;
}

const props = defineProps<Props>();

const iframeRef = ref<HTMLIFrameElement | null>(null);

const fileUrl = computed(() => {
  if (props.data?.file_path && props.data?.session_id) {
    return `/api/v1/sessions/${props.data.session_id}/sandbox-files/${props.data.file_path}`;
  }
  return '';
});

const openFile = () => {
  if (fileUrl.value) {
    window.open(fileUrl.value, '_blank');
  }
};

const downloadFile = async () => {
  if (!fileUrl.value) return;

  try {
    const blob = await getDown(fileUrl.value);
    const blobUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = blobUrl;
    link.download = props.data.file_path || 'report.html';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(blobUrl);
  } catch (error) {
    console.error('[HtmlReport] Download failed:', error);
  }
};

// Render raw HTML directly — the iframe sandbox attribute provides security isolation.
// DOMPurify is intentionally skipped because it strips external CSS/CDN links,
// @import rules, and icon fonts, breaking the visual appearance.
const htmlContent = computed(() => {
  return props.data?.html_content || props.output || '';
});

// Auto-resize iframe to fit content
const onIframeLoad = () => {
  if (!iframeRef.value) return;

  try {
    const iframe = iframeRef.value;
    const doc = iframe.contentDocument || iframe.contentWindow?.document;
    if (!doc) return;

    // Set iframe height to match content
    const resizeObserver = new ResizeObserver(() => {
      const height = doc.documentElement?.scrollHeight || doc.body?.scrollHeight || 600;
      iframe.style.height = `${Math.max(height, 200)}px`;
    });

    resizeObserver.observe(doc.documentElement || doc.body);

    // Initial resize
    const height = doc.documentElement?.scrollHeight || doc.body?.scrollHeight || 600;
    iframe.style.height = `${Math.max(height, 200)}px`;
  } catch (e) {
    // Cross-origin restrictions may prevent access
    console.warn('[HtmlReport] Could not auto-resize iframe:', e);
  }
};

// Watch for content changes and resize
watch(htmlContent, () => {
  setTimeout(() => onIframeLoad(), 100);
});
</script>

<style lang="less" scoped>
.html-file-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;

  .file-info {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;

    .file-icon {
      font-size: 24px;
      color: var(--td-brand-color);
      flex-shrink: 0;
    }

    .file-meta {
      min-width: 0;

      .file-title {
        font-size: 14px;
        font-weight: 500;
        color: var(--td-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .file-name {
        font-size: 12px;
        color: var(--td-text-color-secondary);
        margin-top: 2px;
      }
    }
  }

  .file-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
}

.html-report-container {
  width: 100%;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  overflow: hidden;
  background: #fff;

  .html-report-iframe {
    width: 100%;
    height: 560px;
    border: none;
    display: block;
    background: #fff;
  }
}

// Dark mode support
html[theme-mode="dark"] .html-report-container {
  background: var(--td-bg-color-container);

  .html-report-iframe {
    background: var(--td-bg-color-container);
  }
}
</style>
