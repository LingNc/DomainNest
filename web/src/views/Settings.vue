<!-- web/src/views/Settings.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('settings.title') }}</h2>
      <p class="subtitle">{{ $t('settings.subtitle') }}</p>
    </div>

    <el-row :gutter="20">
      <el-col :xs="24" :lg="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>{{ $t('settings.ddnsToken') }}</span>
              <el-button type="warning" size="small" @click="handleResetToken">{{ $t('settings.resetToken') }}</el-button>
            </div>
          </template>
          <p class="token-label">{{ $t('settings.yourDdnsToken') }}</p>
          <div class="token-row">
            <el-tag size="large" class="token-value">{{ token || $t('common.loading') }}</el-tag>
          </div>

          <el-divider />
          <h4>{{ $t('settings.callbackConfig') }}</h4>
          <p class="hint">{{ $t('settings.callbackHint') }}</p>

          <div class="config-section">
            <div class="config-label">
              <span>URL</span>
              <el-button link type="primary" size="small" @click="copy(callbackUrl)">{{ $t('common.copy') }}</el-button>
            </div>
            <el-input :value="callbackUrl" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>RequestBody</span>
              <el-button link type="primary" size="small" @click="copy(callbackBody)">{{ $t('common.copy') }}</el-button>
            </div>
            <el-input :value="callbackBody" readonly />
          </div>

          <el-divider />
          <h4>{{ $t('settings.webhookConfig') }}</h4>
          <p class="hint">{{ $t('settings.webhookHint') }}</p>

          <div class="config-section">
            <div class="config-label">
              <span>URL</span>
              <el-button link type="primary" size="small" @click="copy(webhookUrl)">{{ $t('common.copy') }}</el-button>
            </div>
            <el-input :value="webhookUrl" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>RequestBody</span>
              <el-button link type="primary" size="small" @click="copy(webhookBody)">{{ $t('common.copy') }}</el-button>
            </div>
            <el-input :value="webhookBody" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>Headers</span>
              <el-button link type="primary" size="small" @click="copy(webhookHeaders)">{{ $t('common.copy') }}</el-button>
            </div>
            <el-input :value="webhookHeaders" readonly />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { ref, computed, onMounted } from 'vue'
import { getProfile, resetToken } from '../api/auth'
import { ElMessage } from 'element-plus'

const { t } = useI18n()

const token = ref('')

// Callback 模式（逐域名更新）
const callbackUrl = computed(() => `${window.location.origin}/api/v1/ddns/callback?token=${token.value}`)
const callbackBody = `{"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}`

// Webhook 模式（聚合更新）
const webhookUrl = computed(() => `${window.location.origin}/api/v1/ddns/webhook`)
const webhookBody = `{"ipv4Addr":"#{ipv4Addr}","ipv4Domains":"#{ipv4Domains}","ipv6Addr":"#{ipv6Addr}","ipv6Domains":"#{ipv6Domains}"}`
const webhookHeaders = computed(() => `Authorization: Bearer ${token.value}`)

const loadProfile = async () => {
  const res = await getProfile()
  token.value = res.data.ddns_token
}

const handleResetToken = async () => {
  const res = await resetToken()
  token.value = res.data.token
  ElMessage.success(t('settings.tokenReset'))
}

const copy = (text) => {
  navigator.clipboard.writeText(text)
  ElMessage.success(t('settings.copiedToClipboard'))
}

onMounted(loadProfile)
</script>

<style scoped>
.page-header {
  margin-bottom: 20px;
}
.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  margin-bottom: 4px;
}
.subtitle {
  color: #909399;
  font-size: 14px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.token-label {
  color: #606266;
  margin-bottom: 8px;
}
.token-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.token-value {
  font-family: monospace;
  font-size: 14px;
}
.hint {
  color: #909399;
  font-size: 13px;
  margin-bottom: 16px;
}
.config-section {
  margin-bottom: 14px;
}
.config-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #303133;
}
</style>
