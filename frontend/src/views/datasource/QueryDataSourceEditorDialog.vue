<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  createQueryDataSource,
  updateQueryDataSource,
  validateQueryDataSourceConnection,
  type QueryDataSource,
} from '@/api/query-datasource'
import DataSourceTypeIcon from '@/views/knowledge/settings/DataSourceTypeIcon.vue'

const props = defineProps<{
  dataSource: QueryDataSource | null
}>()

const visible = defineModel<boolean>('visible', { default: false })
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()

const isEdit = computed(() => !!props.dataSource)
const submitting = ref(false)

// 连接测试
const testing = ref(false)
const testResult = ref<'success' | 'error' | ''>('')
const testErrorMsg = ref('')

// 表单数据
const form = ref({
  name: '',
  type: 'mysql',
  config: {
    host: '',
    port: 3306,
    username: '',
    password: '',
    database: '',
    charset: 'utf8mb4',
    file_path: '',
    sslmode: 'disable',
  },
  description: '',
})

// 数据库类型选项
const dbTypes = [
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'sqlite', label: 'SQLite' },
]

// 当前数据库类型的字段配置
const currentTypeFields = computed(() => {
  switch (form.value.type) {
    case 'mysql':
      return [
        { key: 'host', label: t('datasource.field.host'), placeholder: '127.0.0.1', required: true },
        { key: 'port', label: t('datasource.field.port'), placeholder: '3306', required: true, type: 'number' },
        { key: 'username', label: t('datasource.field.username'), placeholder: 'root', required: true },
        { key: 'password', label: t('datasource.field.password'), placeholder: '', secret: true, required: true },
        { key: 'database', label: t('datasource.field.database'), placeholder: 'mydb', required: true },
        { key: 'charset', label: t('datasource.field.charset'), placeholder: 'utf8mb4', default: 'utf8mb4' },
      ]
    case 'postgresql':
      return [
        { key: 'host', label: t('datasource.field.host'), placeholder: '127.0.0.1', required: true },
        { key: 'port', label: t('datasource.field.port'), placeholder: '5432', required: true, type: 'number' },
        { key: 'username', label: t('datasource.field.username'), placeholder: 'postgres', required: true },
        { key: 'password', label: t('datasource.field.password'), placeholder: '', secret: true, required: true },
        { key: 'database', label: t('datasource.field.database'), placeholder: 'mydb', required: true },
        { key: 'sslmode', label: t('datasource.field.sslmode'), placeholder: 'disable', default: 'disable' },
      ]
    case 'sqlite':
      return [
        { key: 'file_path', label: t('datasource.field.filePath'), placeholder: '/path/to/database.db', required: true },
      ]
    default:
      return []
  }
})

// 监听对话框打开
watch(visible, (v) => {
  if (!v) return
  
  testResult.value = ''
  testErrorMsg.value = ''
  
  if (isEdit.value && props.dataSource) {
    form.value = {
      name: props.dataSource.name,
      type: props.dataSource.type,
      config: {
        host: props.dataSource.config?.host || '',
        port: props.dataSource.config?.port || 3306,
        username: props.dataSource.config?.username || '',
        password: props.dataSource.config?.password || '',
        database: props.dataSource.config?.database || '',
        charset: props.dataSource.config?.charset || 'utf8mb4',
        file_path: props.dataSource.config?.file_path || '',
        sslmode: props.dataSource.config?.sslmode || 'disable',
      },
      description: props.dataSource.description || '',
    }
  } else {
    form.value = {
      name: '',
      type: 'mysql',
      config: {
        host: '',
        port: 3306,
        username: '',
        password: '',
        database: '',
        charset: 'utf8mb4',
        file_path: '',
        sslmode: 'disable',
      },
      description: '',
    }
  }
})

// 测试连接
const handleTestConnection = async () => {
  // 验证必填字段
  const fields = currentTypeFields.value.filter(f => f.required)
  for (const field of fields) {
    const value = form.value.config[field.key as keyof typeof form.value.config]
    if (!value && value !== 0) {
      MessagePlugin.warning(`${field.label} ${t('datasource.isRequired')}`)
      return
    }
  }

  testing.value = true
  testResult.value = ''
  testErrorMsg.value = ''

  try {
    await validateQueryDataSourceConnection({
      name: form.value.name,
      type: form.value.type,
      config: form.value.config,
    })
    testResult.value = 'success'
    MessagePlugin.success(t('datasource.testSuccess'))
  } catch (e: any) {
    testResult.value = 'error'
    testErrorMsg.value = e?.message || e?.error || ''
    MessagePlugin.error(t('datasource.testFailed'))
  }

  testing.value = false
}

// 提交表单
const handleSubmit = async () => {
  // 验证必填字段
  if (!form.value.name) {
    MessagePlugin.warning(t('datasource.nameLabel') + ' ' + t('datasource.isRequired'))
    return
  }

  const fields = currentTypeFields.value.filter(f => f.required)
  for (const field of fields) {
    const value = form.value.config[field.key as keyof typeof form.value.config]
    if (!value && value !== 0) {
      MessagePlugin.warning(`${field.label} ${t('datasource.isRequired')}`)
      return
    }
  }

  submitting.value = true
  try {
    const data = {
      name: form.value.name,
      type: form.value.type,
      config: form.value.config,
      description: form.value.description,
    }

    if (isEdit.value && props.dataSource) {
      await updateQueryDataSource(props.dataSource.id, data)
      MessagePlugin.success(t('datasource.updateSuccess'))
    } else {
      await createQueryDataSource(data)
      MessagePlugin.success(t('datasource.createSuccess'))
    }

    emit('saved')
    visible.value = false
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('datasource.saveFailed'))
  }
  submitting.value = false
}
</script>

<template>
  <t-dialog
    v-model:visible="visible"
    :header="isEdit ? t('datasource.editTitle') : t('datasource.createTitle')"
    :footer="false"
    width="520px"
    destroy-on-close
  >
    <div class="form-item">
      <label class="form-label">{{ t('datasource.nameLabel') }} *</label>
      <t-input v-model="form.name" :placeholder="t('datasource.namePlaceholder')" />
    </div>

    <div class="form-item">
      <label class="form-label">{{ t('datasource.step.selectType') }} *</label>
      <t-select v-model="form.type" :disabled="isEdit">
        <t-option v-for="item in dbTypes" :key="item.value" :value="item.value" :label="item.label" />
      </t-select>
    </div>

    <div v-for="field in currentTypeFields" :key="field.key" class="form-item">
      <label class="form-label">
        {{ field.label }} {{ field.required ? '*' : '' }}
      </label>
      <t-input
        v-model="form.config[field.key as keyof typeof form.config]"
        :placeholder="field.placeholder"
        :type="field.secret ? 'password' : 'text'"
        :number="field.type === 'number'"
      />
    </div>

    <div class="form-item">
      <label class="form-label">{{ t('datasource.description') }}</label>
      <t-textarea
        v-model="form.description"
        :placeholder="t('datasource.descriptionPlaceholder')"
        :autosize="{ minRows: 2, maxRows: 4 }"
      />
    </div>

    <div class="form-actions">
      <t-button variant="outline" :loading="testing" @click="handleTestConnection">
        {{ t('datasource.testConnection') }}
      </t-button>
      <span v-if="testResult === 'success'" class="test-ok">
        <t-icon name="check-circle-filled" size="14px" />
        {{ t('datasource.connected') }}
      </span>
    </div>

    <div v-if="testResult === 'error'" class="test-error-box">
      <t-icon name="error-circle-filled" size="16px" />
      <div class="test-error-content">
        <span class="test-error-title">{{ t('datasource.connectionFailed') }}</span>
        <span v-if="testErrorMsg" class="test-error-detail">{{ testErrorMsg }}</span>
      </div>
    </div>

    <div class="dialog-footer">
      <t-button variant="outline" @click="visible = false">{{ t('datasource.back') }}</t-button>
      <t-button theme="primary" :loading="submitting" @click="handleSubmit">
        {{ isEdit ? t('datasource.save') : t('datasource.createAndSync') }}
      </t-button>
    </div>
  </t-dialog>
</template>

<style scoped>
.form-item {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
  color: var(--td-text-color-primary);
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.test-ok {
  color: var(--td-success-color);
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.test-error-box {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 10px;
  padding: 10px 14px;
  border-radius: 8px;
  background: var(--td-error-color-1);
  color: var(--td-error-color);
  font-size: 13px;
  line-height: 20px;
}

.test-error-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.test-error-title {
  font-weight: 500;
}

.test-error-detail {
  font-size: 12px;
  color: var(--td-error-color);
  opacity: 0.8;
  word-break: break-word;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--td-border-level-2-color);
}
</style>
