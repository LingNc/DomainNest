<!-- web/src/views/Settings.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>系统设置</h2>
      <p class="subtitle">管理您的 DDNS Token 和账号设置</p>
    </div>

    <el-row :gutter="20">
      <el-col :xs="24" :lg="14">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>DDNS Token</span>
              <el-button type="warning" size="small" @click="handleResetToken">重置 Token</el-button>
            </div>
          </template>
          <p class="token-label">您的 DDNS Token：</p>
          <div class="token-row">
            <el-tag size="large" class="token-value">{{ token || '加载中...' }}</el-tag>
          </div>

          <el-divider />
          <h4>ddns-go Callback 配置</h4>
          <p class="hint">在 ddns-go 的「回调」设置中分别填入以下内容：</p>

          <div class="config-section">
            <div class="config-label">
              <span>URL</span>
              <el-button link type="primary" size="small" @click="copy(callbackUrl)">复制</el-button>
            </div>
            <el-input :value="callbackUrl" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>RequestBody</span>
              <el-button link type="primary" size="small" @click="copy(callbackBody)">复制</el-button>
            </div>
            <el-input :value="callbackBody" readonly />
          </div>

          <el-divider />
          <h4>ddns-go Webhook 配置</h4>
          <p class="hint">在 ddns-go 的「Webhook」设置中分别填入以下三项：</p>

          <div class="config-section">
            <div class="config-label">
              <span>URL</span>
              <el-button link type="primary" size="small" @click="copy(webhookUrl)">复制</el-button>
            </div>
            <el-input :value="webhookUrl" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>RequestBody</span>
              <el-button link type="primary" size="small" @click="copy(webhookBody)">复制</el-button>
            </div>
            <el-input :value="webhookBody" readonly />
          </div>

          <div class="config-section">
            <div class="config-label">
              <span>Headers</span>
              <el-button link type="primary" size="small" @click="copy(webhookHeaders)">复制</el-button>
            </div>
            <el-input :value="webhookHeaders" readonly />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="10" style="margin-top:16px">
        <el-card>
          <template #header>修改密码</template>
          <el-form :model="passwordForm" @submit.prevent="handleChangePassword" label-width="80px">
            <el-form-item label="旧密码">
              <el-input v-model="passwordForm.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="passwordForm.new_password" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit">修改密码</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getProfile, resetToken, changePassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const token = ref('')
const passwordForm = ref({ old_password: '', new_password: '' })

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
  ElMessage.success('Token 已重置')
}

const handleChangePassword = async () => {
  await changePassword(passwordForm.value)
  ElMessage.success('密码修改成功')
  passwordForm.value = { old_password: '', new_password: '' }
}

const copy = (text) => {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
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
