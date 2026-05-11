<template>
  <div class="html-report-container">
    <iframe
      ref="iframeRef"
      :srcdoc="htmlContent"
      class="html-report-iframe"
      sandbox="allow-scripts"
      @load="onIframeLoad"
    ></iframe>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import type { HtmlReportData } from '@/types/tool-results';

interface Props {
  data: HtmlReportData;
  output?: string;
}

const props = defineProps<Props>();

const iframeRef = ref<HTMLIFrameElement | null>(null);

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
