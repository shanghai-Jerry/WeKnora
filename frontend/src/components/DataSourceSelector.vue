<template>
  <Teleport to="body">
    <div v-if="visible" class="ds-modal-overlay" @click="close">
      <div class="ds-modal" @click.stop>
        <div class="ds-modal-header">
          <span class="ds-modal-title">Select Data Source</span>
          <button class="ds-modal-close" @click="close">×</button>
        </div>
        <div class="ds-modal-body">
          <div v-if="loading" class="ds-loading">
            <div class="ds-loading-spinner"></div>
            <span>Loading data sources...</span>
          </div>
          <div v-else-if="dataSources.length === 0" class="ds-empty">
            <p>No data sources configured</p>
            <p class="ds-empty-hint">Add a data source in Settings to enable AI SQL queries</p>
          </div>
          <div v-else class="ds-list">
            <div
              v-for="ds in dataSources"
              :key="ds.id"
              class="ds-item"
              :class="{ selected: isSelected(ds.id) }"
              @click="toggleDataSource(ds.id)"
            >
              <div class="ds-item-left">
                <div class="ds-checkbox" :class="{ checked: isSelected(ds.id) }">
                  <svg v-if="isSelected(ds.id)" width="12" height="12" viewBox="0 0 12 12" fill="none">
                    <path d="M10 3L4.5 8.5L2 6" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </div>
                <div class="ds-icon">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
                    <ellipse cx="12" cy="6" rx="8" ry="3" stroke="currentColor" stroke-width="2"/>
                    <path d="M4 6V12C4 13.6569 7.58172 15 12 15C16.4183 15 20 13.6569 20 12V6" stroke="currentColor" stroke-width="2"/>
                    <path d="M4 12V18C4 19.6569 7.58172 21 12 21C16.4183 21 20 19.6569 20 12V12" stroke="currentColor" stroke-width="2"/>
                  </svg>
                </div>
                <div class="ds-info">
                  <span class="ds-name">{{ ds.name }}</span>
                  <span class="ds-type">{{ ds.type }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="ds-modal-footer">
          <button class="ds-btn ds-btn-secondary" @click="clearAll">Clear All</button>
          <button class="ds-btn ds-btn-primary" @click="addNewDataSource">+ Add New Data Source</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import { listQueryDataSources, type QueryDataSource } from '@/api/query-datasource'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const router = useRouter()

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close', 'update:visible'])

const settingsStore = useSettingsStore()

// Local state
const dataSources = ref<QueryDataSource[]>([])
const loading = ref(false)

// Selected data source IDs from store
const selectedDsIds = computed(() => settingsStore.settings.selectedDataSources || [])

const isSelected = (id: string) => selectedDsIds.value.includes(id)

const toggleDataSource = (id: string) => {
  if (isSelected(id)) {
    settingsStore.removeDataSource(id)
  } else {
    settingsStore.addDataSource(id)
  }
}

const clearAll = () => {
  settingsStore.clearDataSources()
}

const addNewDataSource = () => {
  close()
  router.push('/platform/settings')
  setTimeout(() => {
    const event = new CustomEvent('settings-nav', {
      detail: { section: 'datasources' },
    })
    window.dispatchEvent(event)
  }, 100)
}

const close = () => {
  emit('update:visible', false)
  emit('close')
}

const loadDataSources = async () => {
  loading.value = true
  try {
    const res = await listQueryDataSources()
    if (res?.data && Array.isArray(res.data)) {
      // Filter to only active data sources
      dataSources.value = res.data.filter((ds: QueryDataSource) => ds.status === 'active')
    }
  } catch (e) {
    console.error('Failed to load data sources:', e)
    MessagePlugin.error(t('datasource.loadFailed') || 'Failed to load data sources')
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    loadDataSources()
  }
})
</script>

<style scoped lang="less">
.ds-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn 0.15s ease-out;
}

.ds-modal {
  width: 420px;
  max-height: 80vh;
  background: var(--td-bg-color-container, #fff);
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: slideUp 0.2s ease-out;
}

.ds-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--td-component-stroke, #e7e7e7);
}

.ds-modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary, #1f2937);
}

.ds-modal-close {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  font-size: 20px;
  color: var(--td-text-color-secondary, #666);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.12s;

  &:hover {
    background: var(--td-bg-color-secondarycontainer, #f5f5f5);
    color: var(--td-text-color-primary, #333);
  }
}

.ds-modal-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
}

.ds-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--td-text-color-secondary, #666);
  gap: 12px;
}

.ds-loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--td-component-stroke, #e7e7e7);
  border-top-color: var(--td-brand-color, #07c05f);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.ds-empty {
  text-align: center;
  padding: 40px 20px;
  color: var(--td-text-color-secondary, #666);

  p {
    margin: 0 0 8px;
  }
}

.ds-empty-hint {
  font-size: 12px;
  color: var(--td-text-color-placeholder, #999);
}

.ds-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ds-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.12s;

  &:hover {
    background: var(--td-bg-color-secondarycontainer, #f5f5f5);
  }

  &.selected {
    background: var(--td-brand-color-light, #eefdf5);
  }
}

.ds-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.ds-checkbox {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  border: 1.5px solid var(--td-component-border, #e7e7e7);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.12s;

  &.checked {
    background: var(--td-brand-color, #07c05f);
    border-color: var(--td-brand-color, #07c05f);
  }

  svg {
    width: 10px;
    height: 10px;
  }
}

.ds-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--td-brand-color, #07c05f);
}

.ds-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.ds-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary, #1f2937);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ds-type {
  font-size: 11px;
  color: var(--td-text-color-placeholder, #999);
  text-transform: uppercase;
}

.ds-modal-footer {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--td-component-stroke, #e7e7e7);
  background: var(--td-bg-color-secondarycontainer, #fafafa);
}

.ds-btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s;
  border: none;

  &.ds-btn-secondary {
    background: var(--td-bg-color-container, #fff);
    color: var(--td-text-color-secondary, #666);
    border: 1px solid var(--td-component-border, #e7e7e7);

    &:hover {
      background: var(--td-bg-color-secondarycontainer, #f5f5f5);
    }
  }

  &.ds-btn-primary {
    background: var(--td-brand-color, #07c05f);
    color: #fff;
    flex: 1;

    &:hover {
      background: var(--td-brand-color-active, #06a04e);
    }
  }
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
