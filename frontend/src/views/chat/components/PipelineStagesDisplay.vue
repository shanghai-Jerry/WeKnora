<template>
  <div v-if="hasStages" class="pipeline-stages">
    <!-- Header -->
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

      <!-- Step 1: 已理解问题并定位研究方向 -->
      <div v-if="hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot completed">
          <svg viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
        </div>
        <div class="stage-label">已理解问题并定位研究方向</div>
        <div class="stage-body">
          <!-- 统计信息 -->
          <div class="evidence-stats">
            <span class="stat-item">
              <span class="stat-label">检索路径</span>
              <span class="stat-value">{{ intentExplore.analysisPaths.length }} 条</span>
            </span>
            <span class="stat-divider"></span>
          </div>

          <!-- 统一知识网络画布 -->
          <div class="knowledge-network-wrapper">
            <div ref="canvasRef" class="network-canvas" :style="canvasStyle">
              <svg class="network-svg" :viewBox="`0 0 ${layout.contentW} ${layout.h}`" xmlns="http://www.w3.org/2000/svg">
                <!-- 关系连线：极浅灰实线，无箭头 -->
                <g v-for="(edge, idx) in layout.edges" :key="`edge-${idx}`">
                  <line
                    :x1="edge.x1" :y1="edge.y1" :x2="edge.x2" :y2="edge.y2"
                    stroke="#f0f0f0" stroke-width="0.8"
                  />
                  <text
                    v-if="edge.label"
                    :x="edge.midX"
                    :y="edge.midY"
                    text-anchor="middle"
                    dominant-baseline="middle"
                    font-size="9"
                    fill="#bbb"
                    style="pointer-events: none;"
                  >
                    {{ edge.label }}
                  </text>
                </g>

                <!-- 实体到维度的连线 -->
                <g v-for="(dl, idx) in layout.dimLines" :key="`dl-${idx}`">
                  <line
                    :x1="dl.x1" :y1="dl.y1" :x2="dl.x2" :y2="dl.y2"
                    stroke="#f2f2f2" stroke-width="0.6"
                  />
                </g>
              </svg>

              <!-- 实体节点（HTML overlay） -->
              <div
                v-for="node in layout.nodes"
                :key="`node-${node.id}`"
                class="network-entity"
                :class="{ 'is-center': node.isCenter }"
                :style="{ left: `${node.x}px`, top: `${node.y}px` }"
              >
                <span class="entity-text">{{ node.id }}</span>
              </div>

              <!-- 维度节点（HTML overlay） -->
              <div
                v-for="(dim, idx) in layout.dims"
                :key="`dim-${idx}`"
                class="network-dim"
                :style="{ left: `${dim.x}px`, top: `${dim.y}px` }"
              >
                <span class="dim-text">{{ dim.label }}</span>
              </div>
            </div>

            <!-- 超出收起提示 -->
            <div v-if="hiddenEntityCount > 0" class="network-more">
              还有 {{ hiddenEntityCount }} 个关联实体...
            </div>
          </div>
        </div>
      </div>

      <!-- Step 2: 已检索X篇权威内容 -->
      <div v-if="hasReferences || hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot" :class="{ completed: hasReferences }">
          <svg v-if="hasReferences" viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
          <span v-else class="dot-inner"></span>
        </div>
        <div class="stage-label" :class="{ clickable: hasReferences }" @click="hasReferences && toggleCitations()">
          已检索{{ uniqueDocCount }}篇文档
          <t-icon v-if="hasReferences" :name="citationsExpanded ? 'chevron-up' : 'chevron-down'" class="collapse-icon" />
        </div>
        <div class="stage-body">
          <!-- 有引用时：内容可展开/收缩 -->
          <template v-if="hasReferences">
            <div v-show="citationsExpanded">
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

              <div class="citation-list">
                <div v-for="(ref, idx) in visibleReferences" :key="idx" class="citation-item">
                  <p class="citation-title">{{ ref.knowledge_title || ref.knowledge_filename || '未命名文献' }}</p>
                  <p class="citation-source">{{ formatSource(ref) }}</p>
                  <p v-if="ref.content" class="citation-snippet">{{ truncateContent(ref.content) }}</p>
                  <div v-if="idx < visibleReferences.length - 1" class="citation-divider"></div>
                </div>
                <div v-if="knowledgeReferences.length > 5" class="citation-more">
                  还有 {{ knowledgeReferences.length - 5 }} 个分片...
                </div>
              </div>
            </div>
            <div v-show="!citationsExpanded" class="citation-collapsed">
              <span class="collapsed-hint">{{ knowledgeReferences.length }} 个分片，点击展开查看</span>
            </div>
          </template>

          <!-- 无引用时：显示 loading / 空状态 -->
          <template v-else>
            <div class="citation-loading">
              <div v-if="!props.is_completed" class="loading-dots">
                <span></span><span></span><span></span>
              </div>
              <span class="loading-text">{{ props.is_completed ? '未检索到相关文档' : '正在检索文献...' }}</span>
            </div>
          </template>
        </div>
      </div>

      <!-- Step 3: 已完成引用并总结 -->
      <div v-if="hasIntentExplore" class="stage-item evidence-step">
        <div class="timeline-dot completed">
          <svg viewBox="0 0 24 24" fill="currentColor" class="check-icon">
            <path fill-rule="evenodd" d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm13.36-1.814a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z" clip-rule="evenodd" />
          </svg>
        </div>
        <div class="stage-label">已完成引用并总结</div>
      </div>

      <!-- Legacy Pipeline Stages -->
      <template v-if="!hasIntentExplore">
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
        <div v-if="pipelineStages.expansions && pipelineStages.expansions.length > 0" class="stage-item">
          <div class="timeline-dot"></div>
          <div class="stage-label">
            <t-icon name="layers" class="stage-icon" />
            {{ $t('chat.queryExpansion') }}
          </div>
          <div class="stage-body">
            <div class="expansion-list">
              <span v-for="(expansion, idx) in pipelineStages.expansions" :key="idx" class="expansion-tag">
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
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';

interface AnalysisPath {
  path_id: number;
  entity?: string;
  dimensions?: string[] | null;
  reason?: string;
  source_entity?: string;
  target_entity?: string;
  interaction_type?: string;
  mechanistic_link?: string;
  clinical_significance?: string;
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
  knowledge_id?: string;
  knowledge_title?: string;
  knowledge_filename?: string;
  knowledge_source?: string;
  content?: string;
  score?: number;
  match_type?: string;
}

interface PipelineStages {
  queryRewritten?: { originalQuery: string; rewrittenQuery: string };
  retrievalQuery?: string;
  vectorQuery?: string;
  keywordQuery?: string;
  expansions?: string[];
  intentExplore?: IntentExploreData;
}

const props = defineProps<{
  pipelineStages: PipelineStages;
  knowledgeReferences?: ReferenceItem[];
  is_completed?: boolean;
}>();

const expanded = ref(true);
const canvasRef = ref<HTMLDivElement | null>(null);
const canvasSize = ref({ w: 640 });

const MAX_ENTITIES = 8;
const ENTITY_R = 44;
const DIM_R = 18;
const DIM_DIST = 90;   // 维度距实体中心的距离

let ro: ResizeObserver | null = null;

const updateCanvasSize = () => {
  if (!canvasRef.value) return;
  const rect = canvasRef.value.getBoundingClientRect();
  const w = Math.max(320, Math.floor(rect.width));
  if (w > 0) {
    canvasSize.value = { w };
  }
};

const initResizeObserver = () => {
  if (ro) return;
  ro = new ResizeObserver(() => {
    updateCanvasSize();
  });
  if (canvasRef.value) ro.observe(canvasRef.value);
};

onMounted(() => {
  initResizeObserver();
  updateCanvasSize();
});

watch(() => canvasRef.value, (el) => {
  if (el) {
    initResizeObserver();
    updateCanvasSize();
  }
});

// 关键修复：v-show 切换后 DOM 尺寸从 0 恢复，ResizeObserver 可能滞后，
// 因此在 expanded 变为 true 后主动刷新一次尺寸
watch(expanded, async (val) => {
  if (!val) return;
  await nextTick();
  // 给浏览器一次渲染机会后再读尺寸
  requestAnimationFrame(() => {
    updateCanvasSize();
  });
});

onUnmounted(() => { ro?.disconnect(); });

const hasIntentExplore = computed(() => {
  return props.pipelineStages?.intentExplore &&
    props.pipelineStages.intentExplore.analysisPaths &&
    props.pipelineStages.intentExplore.analysisPaths.length > 0;
});

const intentExplore = computed(() => props.pipelineStages?.intentExplore);

const hasReferences = computed(() => props.knowledgeReferences && props.knowledgeReferences.length > 0);
const knowledgeReferences = computed(() => props.knowledgeReferences || []);
const totalReferencesCount = computed(() => {
  if (hasReferences.value) return props.knowledgeReferences!.length;
  return props.pipelineStages?.intentExplore?.totalSearchCount || 0;
});
const uniqueDocCount = computed(() => {
  const uniqueIds = new Set<string>();
  knowledgeReferences.value.forEach((ref) => {
    if (ref.knowledge_id) uniqueIds.add(ref.knowledge_id);
  });
  return uniqueIds.size;
});
const isRetrievalEmpty = computed(() => {
  return props.is_completed && !hasReferences.value && (props.pipelineStages?.retrievalQuery || props.pipelineStages?.vectorQuery || props.pipelineStages?.keywordQuery || props.pipelineStages?.queryRewritten);
});
const visibleReferences = computed(() => knowledgeReferences.value.slice(0, 5));
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

const toggleExpanded = () => { expanded.value = !expanded.value; };

const citationsExpanded = ref(false);
const toggleCitations = () => {
  if (hasReferences.value) citationsExpanded.value = !citationsExpanded.value;
};

/* ============ 图数据转换 ============ */
interface GraphNode {
  id: string;
  dimensions: string[];
  isCenter: boolean;
  x: number;
  y: number;
}

interface GraphEdge {
  from: string;
  to: string;
  x1: number; y1: number; x2: number; y2: number;
  midX: number; midY: number;
  label?: string;
}

interface DimNode {
  label: string;
  parentId: string;
  x: number;
  y: number;
}

interface LayoutResult {
  nodes: GraphNode[];
  dims: DimNode[];
  edges: GraphEdge[];
  dimLines: { x1: number; y1: number; x2: number; y2: number }[];
  h: number;
  contentW: number;
}

// 将 analysisPaths 扁平化为唯一实体图
const allEntities = computed(() => {
  const map = new Map<string, { dimensions: Set<string>; relationCount: number }>();
  intentExplore.value?.analysisPaths.forEach((path) => {
    if (path.entity) {
      const e = map.get(path.entity) || { dimensions: new Set<string>(), relationCount: 0 };
      if (path.dimensions) path.dimensions.forEach((d) => e.dimensions.add(d));
      map.set(path.entity, e);
    }
    if (path.source_entity) {
      const e = map.get(path.source_entity) || { dimensions: new Set<string>(), relationCount: 0 };
      e.relationCount++;
      map.set(path.source_entity, e);
    }
    if (path.target_entity) {
      const e = map.get(path.target_entity) || { dimensions: new Set<string>(), relationCount: 0 };
      e.relationCount++;
      map.set(path.target_entity, e);
    }
  });
  return Array.from(map.entries()).map(([id, data]) => ({
    id,
    dimensions: Array.from(data.dimensions),
    relationCount: data.relationCount,
  }));
});

const hiddenEntityCount = computed(() => Math.max(0, allEntities.value.length - MAX_ENTITIES));

// 选中心实体：关系最多 -> 维度最多 -> 第一个
const centerEntityId = computed(() => {
  const list = allEntities.value;
  if (list.length === 0) return '';
  const sorted = [...list].sort((a, b) => {
    if (b.relationCount !== a.relationCount) return b.relationCount - a.relationCount;
    return b.dimensions.length - a.dimensions.length;
  });
  return sorted[0].id;
});

// 统一布局计算：固定逻辑坐标，自适应容器缩放
const layout = computed<LayoutResult>(() => {
  const entities = allEntities.value.slice(0, MAX_ENTITIES);
  if (entities.length === 0) return { nodes: [], dims: [], edges: [], dimLines: [], h: 220, contentW: 320 };

  const centerId = centerEntityId.value;

  // 节点使用逻辑坐标（中心为原点），不依赖容器像素宽度
  const rawNodes: { id: string; dimensions: string[]; isCenter: boolean; x: number; y: number }[] = [];
  const centerE = entities.find((e) => e.id === centerId) || entities[0];
  rawNodes.push({ id: centerE.id, dimensions: centerE.dimensions, isCenter: true, x: 0, y: 0 });

  const periphery = entities.filter((e) => e.id !== centerId);
  const count = periphery.length;

  // 固定半径，保证节点不重叠、不拥挤
  // 实体半径 40（CSS 中最大实体直径 80px），两实体中心距至少 2*40 + 30 = 110
  // 加上维度标签后，半径需更大
  let radius: number;
  if (count === 0) radius = 0;
  else if (count === 1) radius = 160;
  else if (count === 2) radius = 180;
  else if (count <= 4) radius = 200;
  else if (count <= 6) radius = 220;
  else radius = 240;

  // 起始角度：1个放右侧，2个左右对称，3个均匀，更多圆周
  let startAngle = 0;
  if (count === 1) {
    startAngle = 0; // 右侧
  } else if (count === 2) {
    startAngle = Math.PI; // 180° 开始：左、右
  } else {
    startAngle = -Math.PI / 2 - Math.PI / count;
  }

  periphery.forEach((e, i) => {
    const angle = startAngle + (2 * Math.PI * i) / Math.max(count, 1);
    rawNodes.push({
      id: e.id,
      dimensions: e.dimensions,
      isCenter: false,
      x: radius * Math.cos(angle),
      y: radius * Math.sin(angle),
    });
  });

  // 维度节点：中心全周分布（避开外围实体方向），外围朝外
  // 先收集外围节点的角度方向，供中心实体避让
  const peripheralAngles: number[] = [];
  rawNodes.forEach((n) => {
    if (!n.isCenter && (n.x !== 0 || n.y !== 0)) {
      peripheralAngles.push(Math.atan2(n.y, n.x));
    }
  });

  // 找到避开 blocked 角度的最佳基准角
  const findBestBaseAngle = (blocked: number[], margin: number) => {
    if (blocked.length === 0) return -Math.PI / 2;
    let bestAngle = -Math.PI / 2;
    let bestDist = 0;
    for (let deg = 0; deg < 360; deg += 10) {
      const a = (deg * Math.PI) / 180;
      const minDist = blocked.reduce((min, ba) => {
        let diff = a - ba;
        while (diff > Math.PI) diff -= 2 * Math.PI;
        while (diff < -Math.PI) diff += 2 * Math.PI;
        return Math.min(min, Math.abs(diff));
      }, Infinity);
      if (minDist > bestDist) {
        bestDist = minDist;
        bestAngle = a;
      }
    }
    return bestAngle;
  };

  const rawDims: { label: string; parentId: string; x: number; y: number }[] = [];
  rawNodes.forEach((node) => {
    const dimList = node.dimensions.slice(0, 4);
    if (dimList.length === 0) return;

    let baseAngle: number;
    const spread = Math.min(Math.PI * 0.8, (Math.PI * 0.5) / Math.max(dimList.length - 1, 1));

    if (node.isCenter) {
      const blockMargin = Math.PI / 5;
      baseAngle = findBestBaseAngle(peripheralAngles, blockMargin + spread / 2);
    } else {
      baseAngle = Math.atan2(node.y, node.x);
    }

    dimList.forEach((d, i) => {
      const angle = dimList.length === 1
        ? baseAngle
        : baseAngle - spread / 2 + (spread * i) / (dimList.length - 1);
      rawDims.push({
        label: d,
        parentId: node.id,
        x: node.x + DIM_DIST * Math.cos(angle),
        y: node.y + DIM_DIST * Math.sin(angle),
      });
    });
  });

  // 碰撞规避：迭代排斥重叠的维度节点，并避免连线穿过节点/维度
  const DIM_COLLIDE_R = 40;
  const NODE_COLLIDE_R = ENTITY_R + 8;

  const pointToSegmentDist = (px: number, py: number, ax: number, ay: number, bx: number, by: number) => {
    const abx = bx - ax, aby = by - ay;
    const apx = px - ax, apy = py - ay;
    const ab2 = abx * abx + aby * aby;
    if (ab2 === 0) return Math.sqrt(apx * apx + apy * apy);
    let t = (apx * abx + apy * aby) / ab2;
    t = Math.max(0, Math.min(1, t));
    const cx = ax + t * abx, cy = ay + t * aby;
    const dx = px - cx, dy = py - cy;
    return Math.sqrt(dx * dx + dy * dy);
  };

  const pushDimAwayFromLine = (
    dim: { x: number; y: number; parentId: string },
    ax: number, ay: number, bx: number, by: number,
    minClearance: number,
  ) => {
    const dist = pointToSegmentDist(dim.x, dim.y, ax, ay, bx, by);
    if (dist >= minClearance) return false;
    const push = minClearance - dist + 2;
    const abx = bx - ax, aby = by - ay;
    const abLen = Math.sqrt(abx * abx + aby * aby) || 1;
    const perpX = -aby / abLen;
    const perpY = abx / abLen;
    const side = (dim.x - ax) * aby - (dim.y - ay) * abx;
    const dir = side >= 0 ? 1 : -1;
    dim.x += perpX * push * dir;
    dim.y += perpY * push * dir;
    return true;
  };

  const entityEdgeSegments: { x1: number; y1: number; x2: number; y2: number }[] = [];
  intentExplore.value?.analysisPaths.forEach((path) => {
    if (!path.source_entity || !path.target_entity) return;
    const s = rawNodes.find((n) => n.id === path.source_entity);
    const t = rawNodes.find((n) => n.id === path.target_entity);
    if (!s || !t) return;
    entityEdgeSegments.push({ x1: s.x, y1: s.y, x2: t.x, y2: t.y });
  });

  for (let iter = 0; iter < 30; iter++) {
    let moved = false;
    for (let i = 0; i < rawDims.length; i++) {
      const di = rawDims[i];
      const textWi = Math.max(28, di.label.length * 13 + 10);

      for (let j = i + 1; j < rawDims.length; j++) {
        const dj = rawDims[j];
        const textWj = Math.max(28, dj.label.length * 13 + 10);
        const minDist = (textWi + textWj) / 2 + 4;
        const dx = di.x - dj.x;
        const dy = di.y - dj.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        if (dist < minDist) {
          const push = (minDist - dist) / 2 + 1;
          const nx = dx / dist;
          const ny = dy / dist;
          di.x += nx * push;
          di.y += ny * push;
          dj.x -= nx * push;
          dj.y -= ny * push;
          moved = true;
        }
      }

      rawNodes.forEach((n) => {
        if (n.id === di.parentId) return;
        const dx = di.x - n.x;
        const dy = di.y - n.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const minDist = NODE_COLLIDE_R + DIM_COLLIDE_R / 2;
        if (dist < minDist) {
          const push = minDist - dist + 2;
          di.x += (dx / dist) * push;
          di.y += (dy / dist) * push;
          moved = true;
        }
      });

      const parent = rawNodes.find((n) => n.id === di.parentId);
      if (parent) {
        const dimLineA_x = parent.x, dimLineA_y = parent.y;
        const dimLineB_x = di.x, dimLineB_y = di.y;

        rawNodes.forEach((n) => {
          if (n.id === di.parentId) return;
          const clearance = ENTITY_R + 6;
          const dist = pointToSegmentDist(n.x, n.y, dimLineA_x, dimLineA_y, dimLineB_x, dimLineB_y);
          if (dist < clearance) {
            const dxN = di.x - parent.x, dyN = di.y - parent.y;
            const dLen = Math.sqrt(dxN * dxN + dyN * dyN) || 1;
            const perpX = -dyN / dLen, perpY = dxN / dLen;
            const side = (n.x - parent.x) * dyN - (n.y - parent.y) * dxN;
            const dir = side >= 0 ? 1 : -1;
            di.x += perpX * (clearance - dist + 4) * dir;
            di.y += perpY * (clearance - dist + 4) * dir;
            moved = true;
          }
        });

        for (let j = 0; j < rawDims.length; j++) {
          if (j === i) continue;
          const dj = rawDims[j];
          const clearance = (textWi + Math.max(28, dj.label.length * 13 + 10)) / 2 + 2;
          const dist = pointToSegmentDist(dj.x, dj.y, dimLineA_x, dimLineA_y, dimLineB_x, dimLineB_y);
          if (dist < clearance) {
            const dxN = di.x - parent.x, dyN = di.y - parent.y;
            const dLen = Math.sqrt(dxN * dxN + dyN * dyN) || 1;
            const perpX = -dyN / dLen, perpY = dxN / dLen;
            const side = (dj.x - parent.x) * dyN - (dj.y - parent.y) * dxN;
            const dir = side >= 0 ? 1 : -1;
            di.x += perpX * (clearance - dist + 2) * dir;
            di.y += perpY * (clearance - dist + 2) * dir;
            moved = true;
          }
        }
      }

      entityEdgeSegments.forEach((seg) => {
        const clearance = DIM_COLLIDE_R / 2 + 4;
        if (pushDimAwayFromLine(di, seg.x1, seg.y1, seg.x2, seg.y2, clearance)) {
          moved = true;
        }
      });
    }
    if (!moved) break;
  }

  // 计算内容边界框（考虑文字宽度）
  let minX = 0, maxX = 0, minY = 0, maxY = 0;
  rawNodes.forEach((n) => {
    minX = Math.min(minX, n.x - ENTITY_R - 6);
    maxX = Math.max(maxX, n.x + ENTITY_R + 6);
    minY = Math.min(minY, n.y - ENTITY_R - 6);
    maxY = Math.max(maxY, n.y + ENTITY_R + 6);
  });
  rawDims.forEach((d) => {
    const textW = Math.max(28, d.label.length * 13 + 10); // 估算标签宽度
    minX = Math.min(minX, d.x - textW / 2 - 6);
    maxX = Math.max(maxX, d.x + textW / 2 + 6);
    minY = Math.min(minY, d.y - 12 - 6);
    maxY = Math.max(maxY, d.y + 12 + 6);
  });

  const padX = 28, padY = 30;
  const offsetX = padX - minX;
  const offsetY = padY - minY;
  const contentW = maxX - minX + padX * 2;
  const contentH = maxY - minY + padY * 2;

  // 应用偏移得到最终坐标
  const nodes: GraphNode[] = rawNodes.map((n) => ({ ...n, x: n.x + offsetX, y: n.y + offsetY }));
  const dims: DimNode[] = rawDims.map((d) => ({ ...d, x: d.x + offsetX, y: d.y + offsetY }));
  const nodeMap = new Map(nodes.map((n) => [n.id, n]));

  // 关系边
  const edges: GraphEdge[] = [];
  const seen = new Set<string>();
  intentExplore.value?.analysisPaths.forEach((path) => {
    if (!path.source_entity || !path.target_entity) return;
    const s = nodeMap.get(path.source_entity);
    const t = nodeMap.get(path.target_entity);
    if (!s || !t) return;
    const key = `${path.source_entity}→${path.target_entity}`;
    if (seen.has(key)) return;
    seen.add(key);
    const dx = t.x - s.x;
    const dy = t.y - s.y;
    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
    const nx = dx / dist;
    const ny = dy / dist;
    edges.push({
      from: path.source_entity,
      to: path.target_entity,
      x1: s.x + nx * ENTITY_R,
      y1: s.y + ny * ENTITY_R,
      x2: t.x - nx * ENTITY_R,
      y2: t.y - ny * ENTITY_R,
      midX: (s.x + t.x) / 2,
      midY: (s.y + t.y) / 2,
      label: path.interaction_type,
    });
  });

  // 实体到维度的连线
  const dimLines: { x1: number; y1: number; x2: number; y2: number }[] = [];
  dims.forEach((dim) => {
    const parent = nodeMap.get(dim.parentId);
    if (!parent) return;
    const dx = dim.x - parent.x;
    const dy = dim.y - parent.y;
    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
    dimLines.push({
      x1: parent.x + (dx / dist) * ENTITY_R,
      y1: parent.y + (dy / dist) * ENTITY_R,
      x2: dim.x - (dx / dist) * DIM_R,
      y2: dim.y - (dy / dist) * DIM_R,
    });
  });

  return {
    nodes,
    dims,
    edges,
    dimLines,
    h: Math.max(220, Math.ceil(contentH)),
    contentW: Math.max(320, Math.ceil(contentW)),
  };
});

// 画布宽度和 viewBox 保持一致（逻辑像素 = CSS 像素），
// 这样 SVG 内的连线和 HTML overlay 的节点位置才能 1:1 对齐
const canvasStyle = computed(() => ({
  width: `${layout.value.contentW}px`,
  height: `${layout.value.h}px`,
  margin: '0 auto',
}));

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
}

.stages-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: #f6fef9;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
  border-bottom: 1px solid #e5e7eb;
  &:hover {
    background: #edfdf3;
  }
}

.stages-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #166534;
  .header-icon { width: 18px; height: 18px; color: #22c55e; }
}

.toggle-icon { font-size: 14px; color: var(--td-text-color-secondary); }

.stages-content {
  position: relative;
  padding: 28px 20px 32px 52px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.timeline-track {
  position: absolute;
  left: 32px;
  top: 36px;
  bottom: 36px;
  width: 2px;
  background: linear-gradient(to bottom, var(--td-component-stroke), #d1d5db, var(--td-component-stroke));
  border-radius: 1px;
}

.stage-item {
  position: relative;
  &:not(:last-child) {
    padding-bottom: 8px;
  }
  .timeline-dot {
    position: absolute;
    left: -32px;
    top: 2px;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: white;
    border: 2px solid var(--td-component-stroke);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2;
    transition: all 0.3s ease;
    box-shadow: 0 0 0 4px var(--td-bg-color-container);
    &.completed {
      border-color: #22c55e;
      background: #22c55e;
      .check-icon { width: 14px; height: 14px; color: white; }
    }
    .dot-inner { width: 8px; height: 8px; border-radius: 50%; background: var(--td-component-stroke); }
  }
  .stage-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 16px;
    .stage-icon { font-size: 14px; color: var(--td-brand-color); }
    .collapse-icon { margin-left: auto; font-size: 12px; color: var(--td-text-color-secondary); }
    &.clickable {
      cursor: pointer;
      user-select: none;
      &:hover { color: var(--td-brand-color); }
    }
  }
}

.evidence-step .stage-body {
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 12px;
  border: 1px solid var(--td-component-stroke);
  padding: 20px;
}

.evidence-stats {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding: 10px 14px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #eee;
  .stat-item { display: flex; align-items: center; gap: 6px; }
  .stat-label { font-size: 12px; color: var(--td-text-color-secondary); }
  .stat-value { font-size: 13px; font-weight: 600; color: var(--td-brand-color); }
  .stat-divider { width: 1px; height: 14px; background: var(--td-component-stroke); }
}

/* ===== 统一知识网络画布 ===== */
.knowledge-network-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  overflow-x: auto;
  width: 100%;
  padding: 4px 0;
}

.network-canvas {
  position: relative;
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e8e8e8;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.network-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.network-entity {
  position: absolute;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  transform: translate(-50%, -50%);
  z-index: 2;
  background: #f0f7f7;
  border: 1.5px solid #cce0e0;
  color: #4a7a7a;
  box-shadow: 0 2px 4px rgba(0,0,0,0.05);
  transition: all 0.2s ease;
  &.is-center {
    width: 92px;
    height: 92px;
    background: #e4f2f2;
    border-color: #b0d0d0;
    color: #356060;
    .entity-text { font-size: 13px; font-weight: 700; }
  }
  .entity-text {
    font-size: 11px;
    font-weight: 600;
    line-height: 1.3;
    word-break: break-word;
    padding: 4px;
  }
}

.network-dim {
  position: absolute;
  transform: translate(-50%, -50%);
  z-index: 2;
  background: #fafafa;
  border: 1px solid #e4e4e4;
  color: #a0a0a0;
  border-radius: 10px;
  padding: 3px 10px;
  min-width: 20px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
  .dim-text {
    font-size: 10px;
    line-height: 1.3;
    word-break: break-word;
    white-space: nowrap;
  }
}

.network-more {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

/* ===== Citation ===== */
.source-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 10px;
  margin-bottom: 16px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}
.source-avatars {
  display: flex;
  align-items: center;
  .source-avatar {
    width: 26px; height: 26px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    color: white; font-size: 10px; font-weight: 600;
    border: 2px solid white; margin-left: -6px;
    &:first-child { margin-left: 0; }
  }
}
.source-text { font-size: 12px; color: var(--td-text-color-secondary); }

.citation-list { display: flex; flex-direction: column; gap: 0; }
.citation-item { padding: 14px 4px; position: relative; }
.citation-title { font-size: 13px; font-weight: 500; color: var(--td-text-color-primary); line-height: 1.5; margin-bottom: 6px; }
.citation-source { font-size: 11px; color: var(--td-text-color-secondary); margin-bottom: 6px; }
.citation-snippet { font-size: 11px; color: var(--td-text-color-placeholder); line-height: 1.4; }
.citation-divider { position: absolute; left: 4px; right: 4px; bottom: 0; height: 1px; background: var(--td-component-stroke); }
.citation-more { text-align: center; padding: 12px 0; font-size: 12px; color: var(--td-text-color-secondary); }

.citation-collapsed {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px 0;
  .collapsed-hint {
    font-size: 12px;
    color: var(--td-text-color-secondary);
    background: rgba(0, 0, 0, 0.03);
    padding: 6px 12px;
    border-radius: 6px;
  }
}

.citation-loading {
  display: flex; align-items: center; justify-content: center; gap: 8px; padding: 20px 0;
  .loading-text { font-size: 12px; color: var(--td-text-color-secondary); }
}
.loading-dots {
  display: flex; align-items: center; gap: 4px;
  span { width: 5px; height: 5px; border-radius: 50%; background: var(--td-brand-color); animation: typingBounce 1.4s ease-in-out infinite; }
}

@keyframes typingBounce {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-6px); }
}

/* ===== Legacy ===== */
.query-compare { display: flex; align-items: flex-start; gap: 14px; flex-wrap: wrap; }
.query-item {
  display: flex; flex-direction: column; gap: 6px; padding: 10px 14px;
  border-radius: 8px; font-size: 12px; flex: 1; min-width: 120px; max-width: calc(50% - 20px);
  &.original { background: rgba(0, 0, 0, 0.04); border: 1px solid rgba(0, 0, 0, 0.08); }
  &.rewritten { background: var(--td-brand-color-light); border: 1px solid var(--td-brand-color-focus); }
  .query-tag { font-size: 10px; font-weight: 600; color: var(--td-text-color-placeholder); text-transform: uppercase; letter-spacing: 0.3px; }
  .query-text { color: var(--td-text-color-primary); line-height: 1.5; word-break: break-word; }
}
.arrow-icon { font-size: 14px; color: var(--td-brand-color); flex-shrink: 0; margin-top: 10px; }
.retrieval-query {
  padding: 10px 14px; background: rgba(0, 0, 0, 0.04); border-radius: 8px;
  font-size: 12px; color: var(--td-text-color-primary); line-height: 1.5; word-break: break-word; border: 1px solid rgba(0, 0, 0, 0.08);
  &.vector-query { background: #f9f0ff; border-color: #d3adf7; }
  &.keyword-query { background: #e6f7ff; border-color: #91d5ff; }
}
.expansion-list { display: flex; flex-wrap: wrap; gap: 8px; }
.expansion-tag {
  padding: 5px 12px; background: white; border: 1px solid var(--td-component-stroke);
  border-radius: 8px; font-size: 11px; color: var(--td-text-color-primary); transition: all 0.2s;
  &:hover { border-color: var(--td-brand-color); background: var(--td-brand-color-light); }
}
</style>
