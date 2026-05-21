<!-- web/src/components/RecordFilterBar.vue -->
<template>
  <div class="record-filter-bar">
    <div class="filter-row">
      <el-input
        :model-value="modelValue.host"
        :placeholder="$t('filter.host')"
        clearable
        size="small"
        style="width:140px"
        @update:model-value="v => updateFilter('host', v)"
      />
      <el-select
        :model-value="modelValue.recordType"
        :placeholder="$t('filter.recordType')"
        clearable
        multiple
        collapse-tags
        collapse-tags-tooltip
        size="small"
        style="width:180px"
        @update:model-value="v => updateFilter('recordType', v)"
      >
        <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
      </el-select>
      <el-input
        :model-value="modelValue.value"
        :placeholder="$t('filter.value')"
        clearable
        size="small"
        style="width:160px"
        @update:model-value="v => updateFilter('value', v)"
      />
      <el-select
        :model-value="modelValue.status"
        :placeholder="$t('filter.status')"
        clearable
        size="small"
        style="width:100px"
        @update:model-value="v => updateFilter('status', v)"
      >
        <el-option :label="$t('filter.all')" value="" />
        <el-option :label="$t('filter.enabled')" value="enabled" />
        <el-option :label="$t('filter.disabled')" value="disabled" />
      </el-select>
      <el-select
        :model-value="modelValue.source"
        :placeholder="$t('filter.source')"
        clearable
        size="small"
        style="width:110px"
        @update:model-value="v => updateFilter('source', v)"
      >
        <el-option :label="$t('filter.all')" value="" />
        <el-option :label="$t('filter.platform')" value="platform" />
        <el-option :label="$t('filter.provider')" value="provider" />
      </el-select>
      <el-button size="small" @click="clearAll">{{ $t('filter.clearAll') }}</el-button>
      <el-button size="small" type="primary" @click="showSaveDialog = true" :disabled="!hasActiveFilters">
        {{ $t('filter.savePreset') }}
      </el-button>
      <el-dropdown v-if="presets.length > 0" trigger="click" @command="loadPreset">
        <el-button size="small">{{ $t('filter.loadPreset') }} <el-icon class="el-icon--right"><component :is="'ArrowDown'" /></el-icon></el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-for="p in presets" :key="p.id" :command="p">
              <div class="preset-item">
                <span>{{ p.name }}</span>
                <el-icon class="preset-delete" @click.stop="handleDeletePreset(p.id)"><component :is="'Delete'" /></el-icon>
              </div>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <!-- active filter tags -->
    <div v-if="activeFilterTags.length > 0" class="filter-tags">
      <span class="filter-tags-label">{{ $t('filter.activeFilters') }}:</span>
      <el-tag
        v-for="tag in activeFilterTags"
        :key="tag.key"
        size="small"
        closable
        @close="removeFilter(tag.key)"
      >{{ tag.label }}</el-tag>
    </div>

    <!-- save preset dialog -->
    <el-dialog v-model="showSaveDialog" :title="$t('filter.savePreset')" width="360px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item :label="$t('filter.presetName')">
          <el-input v-model="presetName" :placeholder="$t('filter.presetName')" maxlength="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSaveDialog = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSavePreset">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listFilterPresets, saveFilterPreset, deleteFilterPreset } from '../api/filterPreset'

const props = defineProps({
  modelValue: { type: Object, required: true },
  recordTypes: { type: Array, required: true },
})

const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()

const presets = ref([])
const showSaveDialog = ref(false)
const presetName = ref('')

let debounceTimer = null

const updateFilter = (key, value) => {
  const updated = { ...props.modelValue, [key]: value }
  emit('update:modelValue', updated)
}

const clearAll = () => {
  emit('update:modelValue', { host: '', recordType: [], value: '', status: '', source: '' })
}

const removeFilter = (key) => {
  const defaultVal = key === 'recordType' ? [] : ''
  updateFilter(key, defaultVal)
}

const hasActiveFilters = computed(() => {
  const f = props.modelValue
  return f.host || (f.recordType && f.recordType.length > 0) || f.value || f.status || f.source
})

const activeFilterTags = computed(() => {
  const tags = []
  const f = props.modelValue
  if (f.host) tags.push({ key: 'host', label: `${t('filter.host')}: ${f.host}` })
  if (f.recordType && f.recordType.length > 0) {
    tags.push({ key: 'recordType', label: `${t('filter.recordType')}: ${f.recordType.join(', ')}` })
  }
  if (f.value) tags.push({ key: 'value', label: `${t('filter.value')}: ${f.value}` })
  if (f.status) tags.push({ key: 'status', label: `${t('filter.status')}: ${f.status === 'enabled' ? t('filter.enabled') : t('filter.disabled')}` })
  if (f.source) tags.push({ key: 'source', label: `${t('filter.source')}: ${f.source === 'platform' ? t('filter.platform') : t('filter.provider')}` })
  return tags
})

const loadPresets = async () => {
  try {
    const res = await listFilterPresets()
    presets.value = res.data || []
  } catch { /* ignore */ }
}

const handleSavePreset = async () => {
  if (!presetName.value.trim()) {
    ElMessage.warning(t('filter.presetName'))
    return
  }
  try {
    await saveFilterPreset({ name: presetName.value.trim(), filters: JSON.stringify(props.modelValue) })
    ElMessage.success(t('common.save') + ' ' + t('common.confirm'))
    showSaveDialog.value = false
    presetName.value = ''
    loadPresets()
  } catch { /* error handled by interceptor */ }
}

const loadPreset = (preset) => {
  try {
    const filters = JSON.parse(preset.filters)
    emit('update:modelValue', { host: '', recordType: [], value: '', status: '', source: '', ...filters })
  } catch { /* ignore */ }
}

const handleDeletePreset = async (id) => {
  try {
    await ElMessageBox.confirm(t('filter.deletePreset'), t('common.hint'), { type: 'warning' })
    await deleteFilterPreset(id)
    ElMessage.success(t('common.delete') + ' ' + t('common.confirm'))
    loadPresets()
  } catch { /* cancelled or error */ }
}

onMounted(() => {
  loadPresets()
})
</script>

<style scoped>
.record-filter-bar {
  margin-bottom: 12px;
}
.filter-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.filter-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;
}
.filter-tags-label {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}
.preset-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}
.preset-delete {
  color: #f56c6c;
  cursor: pointer;
}
.preset-delete:hover {
  opacity: 0.8;
}
</style>
