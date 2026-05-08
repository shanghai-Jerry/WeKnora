<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { 
  listQueryDataSources, 
  deleteQueryDataSource, 
  type QueryDataSource 
} from '@/api/query-datasource'
import QueryDataSourceEditorDialog from './QueryDataSourceEditorDialog.vue'
import DataSourceTypeIcon from '@/views/knowledge/settings/DataSourceTypeIcon.vue'

const { t } = useI18n()

// 数据源列表
const dataSources = ref<QueryDataSource[]>([])
const loading = ref(false)

// 编辑对话框
const showEditor = ref(false)
const editingDataSource = ref<QueryDataSource | null>(null)

// 获取数据源列表
const fetchDataSources = async () => {
  loading.value = true
  try {
    const res = await listQueryDataSources()
    dataSources.value = res?.data || []
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('datasource.loadFailed'))
  }
  loading.value = false
}

// 格式化时间
const formatTime = (time: string | null) => {
  if (!time) return '-'
  const date = new Date(time)
  return date.toLocaleString()
}

// 创建数据源
const handleCreate = () => {
  editingDataSource.value = null
  showEditor.value = true
}

// 编辑数据源
const handleEdit = (ds: QueryDataSource) => {
  editingDataSource.value = ds
  showEditor.value = true
}

// 删除数据源
const handleDelete = async (ds: QueryDataSource) => {
  const confirm = DialogPlugin.confirm({
    header: t('datasource.delete'),
    body: t('datasource.deleteConfirm'),
    confirmBtn: {
      content: t('datasource.delete'),
      theme: 'danger',
    },
    onConfirm: async () => {
      try {
        await deleteQueryDataSource(ds.id)
        MessagePlugin.success(t('datasource.deleteSuccess'))
        fetchDataSources()
      } catch (e: any) {
        MessagePlugin.error(e?.message || t('datasource.deleteFailed'))
      }
    },
  })
}

// 保存成功回调
const handleSaved = () => {
  fetchDataSources()
}

// 状态标签样式
const getStatusClass = (status: string) => {
  switch (status) {
    case 'active': return 'status-active'
    case 'inactive': return 'status-inactive'
    case 'error': return 'status-error'
    default: return ''
  }
}

onMounted(() => {
  fetchDataSources()
})
</script>

<template>
  <div class="ds-list-page">
    <div class="ds-list-header">
      <div class="ds-list-title">
        <h2>{{ t('datasource.title') }}</h2>
        <p class="ds-list-desc">{{ t('datasource.description') }}</p>
      </div>
      <t-button theme="primary" @click="handleCreate">
        <template #icon><t-icon name="add" /></template>
        {{ t('datasource.add') }}
      </t-button>
    </div>

    <div v-if="loading" class="ds-list-loading">
      <t-loading />
    </div>

    <div v-else-if="dataSources.length === 0" class="ds-list-empty">
      <t-icon name="info-circle" size="48px" style="color: var(--td-text-color-placeholder)" />
      <p>{{ t('datasource.empty') }}</p>
      <t-button theme="primary" variant="outline" @click="handleCreate">
        {{ t('datasource.addFirst') }}
      </t-button>
    </div>

    <div v-else class="ds-list-grid">
      <div 
        v-for="ds in dataSources" 
        :key="ds.id" 
        class="ds-card"
      >
        <div class="ds-card-header">
          <div class="ds-card-icon">
            <DataSourceTypeIcon :type="ds.type" :size="32" />
          </div>
          <div class="ds-card-title">
            <h3>{{ ds.name }}</h3>
            <span class="ds-card-type">{{ t(`datasource.connector.${ds.type}`) }}</span>
          </div>
          <div class="ds-card-status">
            <span :class="['status-badge', getStatusClass(ds.status)]">
              {{ t(`datasource.status.${ds.status}`) }}
            </span>
          </div>
        </div>

        <div class="ds-card-content">
          <div class="ds-card-info">
            <div v-if="ds.type === 'mysql' || ds.type === 'postgresql'" class="ds-info-row">
              <span class="ds-info-label">Host:</span>
              <span class="ds-info-value">{{ ds.config?.host || '-' }}</span>
            </div>
            <div v-if="ds.type === 'mysql' || ds.type === 'postgresql'" class="ds-info-row">
              <span class="ds-info-label">Port:</span>
              <span class="ds-info-value">{{ ds.config?.port || '-' }}</span>
            </div>
            <div v-if="ds.type === 'mysql' || ds.type === 'postgresql'" class="ds-info-row">
              <span class="ds-info-label">Database:</span>
              <span class="ds-info-value">{{ ds.config?.database || '-' }}</span>
            </div>
            <div v-if="ds.type === 'sqlite'" class="ds-info-row">
              <span class="ds-info-label">File:</span>
              <span class="ds-info-value">{{ ds.config?.file_path || '-' }}</span>
            </div>
            <div class="ds-info-row">
              <span class="ds-info-label">{{ t('datasource.createdAt') }}:</span>
              <span class="ds-info-value">{{ formatTime(ds.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="ds-card-actions">
          <t-button 
            variant="outline" 
            size="small"
            @click="handleEdit(ds)"
          >
            <template #icon><t-icon name="edit" /></template>
            {{ t('datasource.edit') }}
          </t-button>
          <t-button 
            theme="danger" 
            variant="outline" 
            size="small"
            @click="handleDelete(ds)"
          >
            <template #icon><t-icon name="delete" /></template>
            {{ t('datasource.delete') }}
          </t-button>
        </div>
      </div>
    </div>

    <QueryDataSourceEditorDialog
      v-model:visible="showEditor"
      :data-source="editingDataSource"
      @saved="handleSaved"
    />
  </div>
</template>

<style scoped>
.ds-list-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.ds-list-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.ds-list-title h2 {
  margin: 0 0 4px;
  font-size: 20px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.ds-list-desc {
  margin: 0;
  font-size: 14px;
  color: var(--td-text-color-secondary);
}

.ds-list-loading {
  display: flex;
  justify-content: center;
  padding: 60px 0;
}

.ds-list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  gap: 16px;
}

.ds-list-empty p {
  margin: 0;
  font-size: 14px;
  color: var(--td-text-color-secondary);
}

.ds-list-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.ds-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-2-color);
  border-radius: 8px;
  padding: 16px;
  transition: box-shadow 0.2s;
}

.ds-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.ds-card-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.ds-card-icon {
  flex-shrink: 0;
}

.ds-card-title {
  flex: 1;
  min-width: 0;
}

.ds-card-title h3 {
  margin: 0 0 2px;
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ds-card-type {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.ds-card-status {
  flex-shrink: 0;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-active {
  background: var(--td-success-color-1);
  color: var(--td-success-color);
}

.status-inactive {
  background: var(--td-warning-color-1);
  color: var(--td-warning-color);
}

.status-error {
  background: var(--td-error-color-1);
  color: var(--td-error-color);
}

.ds-card-content {
  margin-bottom: 12px;
}

.ds-card-info {
  font-size: 13px;
}

.ds-info-row {
  display: flex;
  margin-bottom: 4px;
}

.ds-info-label {
  color: var(--td-text-color-secondary);
  margin-right: 8px;
}

.ds-info-value {
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ds-card-actions {
  display: flex;
  gap: 8px;
}
</style>
