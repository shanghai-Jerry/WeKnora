<template>
  <Teleport to="body">
    <div v-if="visible" class="qp-modal-overlay" @click="close">
      <div class="qp-modal" @click.stop>
        <div class="qp-modal-header">
          <span class="qp-modal-title">{{ t('queryParam.title') }}</span>
          <button class="qp-modal-close" @click="close">&times;</button>
        </div>
        <div class="qp-modal-body">
          <div v-if="params.length === 0" class="qp-empty">
            <p>{{ t('queryParam.empty') }}</p>
            <p class="qp-empty-hint">{{ t('queryParam.emptyHint') }}</p>
          </div>
          <div v-else class="qp-list">
            <div v-for="(param, index) in params" :key="index" class="qp-item">
              <div class="qp-item-fields">
                <div class="qp-field">
                  <label>{{ t('queryParam.fieldName') }}</label>
                  <input
                    :value="param.field_name"
                    @input="updateField(index, 'field_name', ($event.target as HTMLInputElement).value)"
                    :placeholder="t('queryParam.fieldNamePlaceholder')"
                    class="qp-input"
                  />
                </div>
                <div class="qp-field">
                  <label>{{ t('queryParam.fieldValue') }}</label>
                  <input
                    :value="param.field_value"
                    @input="updateField(index, 'field_value', ($event.target as HTMLInputElement).value)"
                    :placeholder="t('queryParam.fieldValuePlaceholder')"
                    class="qp-input"
                  />
                </div>
                <div class="qp-field">
                  <label>{{ t('queryParam.fieldDescription') }}</label>
                  <input
                    :value="param.field_description"
                    @input="updateField(index, 'field_description', ($event.target as HTMLInputElement).value)"
                    :placeholder="t('queryParam.fieldDescriptionPlaceholder')"
                    class="qp-input"
                  />
                </div>
              </div>
              <button class="qp-remove-btn" @click="removeParam(index)" :title="t('queryParam.remove')">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M10.5 3.5L3.5 10.5M3.5 3.5L10.5 10.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <div class="qp-modal-footer">
          <button class="qp-btn qp-btn-secondary" @click="clearAll">{{ t('queryParam.clearAll') }}</button>
          <button class="qp-btn qp-btn-primary" @click="addParam">+ {{ t('queryParam.add') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore, type QueryParamItem } from '@/stores/settings'

const { t } = useI18n()
const settingsStore = useSettingsStore()

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close', 'update:visible'])

const params = ref<QueryParamItem[]>([])

watch(() => props.visible, (v) => {
  if (v) {
    // Load from store when opening
    params.value = (settingsStore.settings.queryParams || []).map(p => ({ ...p }))
  }
})

const updateField = (index: number, field: keyof QueryParamItem, value: string) => {
  params.value[index] = { ...params.value[index], [field]: value }
  settingsStore.updateQueryParam(index, params.value[index])
}

const addParam = () => {
  const newParam: QueryParamItem = { field_name: '', field_value: '', field_description: '' }
  params.value.push(newParam)
  settingsStore.addQueryParam(newParam)
}

const removeParam = (index: number) => {
  params.value.splice(index, 1)
  settingsStore.removeQueryParam(index)
}

const clearAll = () => {
  params.value = []
  settingsStore.clearQueryParams()
}

const close = () => {
  emit('update:visible', false)
  emit('close')
}
</script>

<style scoped lang="less">
.qp-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn 0.15s ease-out;
}

.qp-modal {
  width: 520px;
  max-height: 80vh;
  background: var(--td-bg-color-container, #fff);
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: slideUp 0.2s ease-out;
}

.qp-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--td-component-stroke, #e7e7e7);
}

.qp-modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary, #1f2937);
}

.qp-modal-close {
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

.qp-modal-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 16px;
}

.qp-empty {
  text-align: center;
  padding: 40px 20px;
  color: var(--td-text-color-secondary, #666);

  p {
    margin: 0 0 8px;
  }
}

.qp-empty-hint {
  font-size: 12px;
  color: var(--td-text-color-placeholder, #999);
}

.qp-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.qp-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer, #fafafa);
  border: 1px solid var(--td-component-stroke, #e7e7e7);
}

.qp-item-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.qp-field {
  display: flex;
  flex-direction: column;
  gap: 4px;

  label {
    font-size: 11px;
    font-weight: 500;
    color: var(--td-text-color-secondary, #666);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
}

.qp-input {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid var(--td-component-border, #e7e7e7);
  border-radius: 6px;
  font-size: 13px;
  color: var(--td-text-color-primary, #1f2937);
  background: var(--td-bg-color-container, #fff);
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;

  &:focus {
    border-color: var(--td-brand-color, #07c05f);
  }

  &::placeholder {
    color: var(--td-text-color-placeholder, #999);
  }
}

.qp-remove-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  color: var(--td-text-color-placeholder, #999);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 20px;
  transition: all 0.12s;

  &:hover {
    background: var(--td-error-color-1, #fff0f0);
    color: var(--td-error-color, #e34d59);
  }
}

.qp-modal-footer {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--td-component-stroke, #e7e7e7);
  background: var(--td-bg-color-secondarycontainer, #fafafa);
}

.qp-btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s;
  border: none;

  &.qp-btn-secondary {
    background: var(--td-bg-color-container, #fff);
    color: var(--td-text-color-secondary, #666);
    border: 1px solid var(--td-component-border, #e7e7e7);

    &:hover {
      background: var(--td-bg-color-secondarycontainer, #f5f5f5);
    }
  }

  &.qp-btn-primary {
    background: var(--td-brand-color, #07c05f);
    color: #fff;

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
</style>
