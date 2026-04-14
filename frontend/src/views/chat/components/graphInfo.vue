<template>
    <div class="graph-info" v-if="graphData && (graphData.nodes?.length || graphData.relations?.length)">
        <div class="graph-header" @click="toggleBox">
            <div class="graph-title">
                <t-icon name="link" size="16px" class="graph-icon" />
                <span>{{ headerText }}</span>
            </div>
            <div class="graph-show-icon">
                <t-icon :name="showBox ? 'chevron-up' : 'chevron-down'" />
            </div>
        </div>
        <div class="graph-box" v-show="showBox">
            <!-- Nodes section -->
            <div class="graph-section" v-if="graphData.nodes?.length">
                <div class="graph-section-title">{{ $t('chat.graphNodes') }}</div>
                <div class="graph-item-list">
                    <div v-for="(node, index) in graphData.nodes" :key="'node-' + index" class="graph-item graph-node">
                        <t-icon name="circle" size="6px" class="graph-node-dot" />
                        <span class="graph-node-name" :title="node.name">{{ node.name }}</span>
                        <span v-if="node.attributes?.length" class="graph-node-attrs" :title="node.attributes.join(', ')">
                            ({{ node.attributes.join(', ') }})
                        </span>
                    </div>
                </div>
            </div>
            <!-- Relations section -->
            <div class="graph-section" v-if="graphData.relations?.length">
                <div class="graph-section-title">{{ $t('chat.graphRelations') }}</div>
                <div class="graph-item-list">
                    <div v-for="(rel, index) in graphData.relations" :key="'rel-' + index" class="graph-item graph-relation">
                        <span class="graph-rel-node">{{ rel.node1 }}</span>
                        <span class="graph-rel-type">{{ rel.type }}</span>
                        <span class="graph-rel-node">{{ rel.node2 }}</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    session: {
        type: Object,
        required: false
    }
});

const showBox = ref(false);

const toggleBox = () => {
    showBox.value = !showBox.value;
};

const graphData = computed(() => {
    return props.session?.graph_data || null;
});

const headerText = computed(() => {
    const nodeCount = graphData.value?.nodes?.length ?? 0;
    const relCount = graphData.value?.relations?.length ?? 0;
    return t('chat.graphHeader', { nodeCount, relCount });
});
</script>
<style lang="less" scoped>
.graph-info {
    display: flex;
    flex-direction: column;
    font-size: 12px;
    width: 100%;
    border-radius: 8px;
    background-color: var(--td-bg-color-container);
    border: .5px solid var(--td-component-stroke);
    box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
    overflow: hidden;
    box-sizing: border-box;
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    margin-bottom: 8px;

    .graph-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 6px 14px;
        color: var(--td-text-color-primary);
        font-weight: 500;

        .graph-title {
            display: flex;
            align-items: center;

            .graph-icon {
                color: var(--td-brand-color);
                margin-right: 8px;
            }

            span {
                white-space: nowrap;
                font-size: 12px;
            }
        }

        .graph-show-icon {
            font-size: 14px;
            padding: 0 2px 1px 2px;
            color: var(--td-brand-color);
        }
    }

    .graph-header:hover {
        background-color: rgba(7, 192, 95, 0.04);
        cursor: pointer;
    }

    .graph-box {
        padding: 4px 14px 8px 14px;
        border-top: 1px solid var(--td-bg-color-secondarycontainer);
    }
}

.graph-section {
    margin-top: 4px;

    .graph-section-title {
        color: var(--td-text-color-secondary);
        font-size: 11px;
        font-weight: 500;
        margin-bottom: 4px;
        padding-left: 2px;
    }
}

.graph-item-list {
    max-height: 200px;
    overflow-y: auto;
}

.graph-item {
    padding: 2px 4px;
    border-radius: 4px;
    transition: background-color 0.15s ease;

    &:hover {
        background-color: rgba(7, 192, 95, 0.04);
    }
}

.graph-node {
    display: flex;
    align-items: center;
    line-height: 20px;

    .graph-node-dot {
        color: var(--td-brand-color);
        flex-shrink: 0;
        margin-right: 6px;
    }

    .graph-node-name {
        color: var(--td-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 200px;
    }

    .graph-node-attrs {
        color: var(--td-text-color-placeholder);
        font-size: 11px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-left: 4px;
        max-width: 180px;
    }
}

.graph-relation {
    display: flex;
    align-items: center;
    line-height: 20px;
    flex-wrap: wrap;

    .graph-rel-node {
        color: var(--td-brand-color);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 150px;
    }

    .graph-rel-type {
        color: var(--td-text-color-placeholder);
        font-size: 11px;
        margin: 0 6px;
        white-space: nowrap;
    }
}
</style>
