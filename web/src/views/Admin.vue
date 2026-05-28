<!-- web/src/views/Admin.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('admin.title') }}</h2>
      <p class="subtitle">{{ $t('admin.subtitle') }}</p>
    </div>

    <el-card>
      <el-tabs v-model="activeTab">
        <!-- 用户管理 -->
        <el-tab-pane :label="$t('admin.userManagement')" name="users">
          <div v-if="selectedUsers.length > 0" style="margin-bottom:12px;display:flex;align-items:center;gap:8px">
            <el-tag>{{ $t('admin.selectedCount', { count: selectedUsers.length }) }}</el-tag>
            <el-button type="primary" size="small" @click="openBatchGrant">{{ $t('admin.batchGrantInvite') }}</el-button>
            <el-button type="danger" size="small" @click="openBatchRevoke">{{ $t('admin.batchRevokeInvite') }}</el-button>
          </div>
          <el-table :data="users" stripe v-loading="loading" style="width:100%" @selection-change="handleSelectionChange">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="id" :label="$t('admin.id')" width="60" />
            <el-table-column :label="$t('admin.avatar')" width="60">
              <template #default="{ row }">
                <el-avatar v-if="row.avatar" :src="row.avatar" :size="32" />
                <el-avatar v-else :size="32">{{ (row.username || '?')[0]?.toUpperCase() }}</el-avatar>
              </template>
            </el-table-column>
            <el-table-column prop="username" :label="$t('common.username')" min-width="100" />
            <el-table-column prop="email" :label="$t('admin.email')" min-width="140" show-overflow-tooltip />
            <el-table-column prop="role" :label="$t('admin.role')" width="90">
              <template #default="{ row }">
                <el-tag :type="row.is_super_admin ? 'warning' : row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.is_super_admin ? $t('admin.superAdmin') : row.role === 'admin' ? $t('common.admin') : $t('admin.normalUser') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="$t('common.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
                  {{ row.status === 1 ? $t('admin.statusNormal') : $t('admin.statusDisabled') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="invite_limit" :label="$t('admin.inviteLimit')" width="90" />
            <el-table-column prop="invite_count" :label="$t('admin.invitedCount')" width="80" />
            <el-table-column prop="created_at" :label="$t('admin.registerTime')" width="170" />
            <el-table-column :label="$t('common.action')" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="success" size="small" :disabled="row.is_super_admin && !auth.isSuperAdmin" @click="openGrantDialog(row)">{{ $t('admin.grantInviteAction') }}</el-button>
                <el-button link type="danger" size="small" :disabled="row.is_super_admin && !auth.isSuperAdmin" @click="openRevokeDialog(row)">{{ $t('admin.revokeInviteAction') }}</el-button>
                <el-button link type="primary" size="small" :disabled="row.is_super_admin || (row.is_admin && !auth.isSuperAdmin)" @click="openEditUser(row)">{{ $t('common.edit') }}</el-button>
                <el-button link type="warning" size="small" :disabled="row.is_super_admin && row.id !== auth.user?.id" @click="openResetPwd(row)">{{ $t('admin.resetPassword') }}</el-button>
                <el-button v-if="row.status === 1" link type="danger" size="small" :disabled="row.is_super_admin || (row.is_admin && !auth.isSuperAdmin)" @click="handleDisable(row)">{{ $t('admin.disable') }}</el-button>
                <el-button v-else link type="success" size="small" @click="handleEnable(row)">{{ $t('admin.enable') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 域名管理 -->
        <el-tab-pane :label="$t('admin.domainManagement')" name="domains">
          <div v-if="!loadingAdminProviders && adminProviders.length === 0" style="padding:40px 0">
            <el-empty :description="$t('admin.noProvidersHint')" :image-size="60" />
          </div>
          <template v-else>
          <div class="domain-actions">
            <div style="display:flex;gap:8px;align-items:flex-end;flex-wrap:wrap">
              <el-select v-model="selectedProviderId" :placeholder="$t('admin.selectProvider')" style="width:200px" size="default">
                <el-option v-for="p in adminProviders" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
              <el-select v-model="selectedDomain" :placeholder="$t('admin.selectDomain')" style="width:240px" size="default" :loading="loadingAdminDomains" :disabled="!selectedProviderId">
                <el-option v-for="d in adminDomains" :key="d.domain_name" :label="getDomainLabel(d)" :value="d.domain_name" />
              </el-select>
              <el-button type="primary" size="default" @click="handleCreateRoot">{{ $t('admin.createRootDomain') }}</el-button>
            </div>
          </div>

          <el-input v-model="domainTreeSearch" :placeholder="$t('admin.domainTreeSearch')" clearable style="margin-bottom:12px;width:300px" />

          <div v-if="selectedDomains.length > 0" style="margin-bottom:12px;display:flex;gap:8px;align-items:center">
            <el-tag>{{ $t('admin.selectedCount', { count: selectedDomains.length }) }}</el-tag>
            <el-button type="danger" size="small" :loading="batchDeleteLoading" @click="handleBatchDeleteDomains">
              {{ $t('admin.batchDeleteDomains') }}
            </el-button>
          </div>

          <div class="domain-tree-layout">
            <el-table :data="filteredDomainTree" stripe v-loading="loading" style="width:100%"
              row-key="id" :tree-props="{ children: 'children' }"
              @selection-change="handleDomainSelectionChange"
            >
              <el-table-column type="selection" width="40" :selectable="() => true" />
              <el-table-column prop="full_domain" :label="$t('admin.fullDomain')" min-width="200" show-overflow-tooltip />
              <el-table-column :label="$t('admin.currentOwner')" min-width="130">
                <template #default="{ row }">
                  <div style="display:flex;align-items:center;gap:6px">
                    <el-avatar v-if="row.owner?.avatar" :src="row.owner.avatar" :size="22" />
                    <el-avatar v-else :size="22">{{ (row.owner?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                    <span style="font-size:13px">{{ row.owner?.username || row.owner_id }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="$t('common.status')" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                    {{ row.status === 'active' ? $t('admin.statusNormal') : $t('dashboard.archived') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="$t('admin.permissionsCount', { count: '' })" width="80">
                <template #default="{ row }">
                  <el-tag size="small" type="info">{{ row.permission_count ?? 0 }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" />
              <el-table-column :label="$t('common.action')" width="80" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click.stop="$router.push(`/admin/domains/${row.id}`)">{{ $t('admin.viewDetail') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          </template>
        </el-tab-pane>

        <!-- 操作日志 -->
        <el-tab-pane :label="$t('admin.operationLogs')" name="logs">
          <div class="filter-bar">
            <el-select v-model="logFilters.user_id" :placeholder="$t('admin.user')" clearable filterable size="small" style="width:130px">
              <el-option v-for="u in users" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="u.avatar" :src="u.avatar" :size="20" />
                  <el-avatar v-else :size="20">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ u.nickname || u.username }}</span>
                </div>
              </el-option>
            </el-select>
            <el-select v-model="logFilters.target_user_id" :placeholder="$t('admin.targetUserLabel')" clearable filterable size="small" style="width:130px">
              <el-option v-for="u in users" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="u.avatar" :src="u.avatar" :size="20" />
                  <el-avatar v-else :size="20">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ u.nickname || u.username }}</span>
                </div>
              </el-option>
            </el-select>
            <el-input v-model="logFilters.q" :placeholder="$t('myLogs.searchPlaceholder')" clearable size="small" style="width:120px" />
            <el-input v-model="logFilters.q_exclude" :placeholder="$t('myLogs.excludePlaceholder')" clearable size="small" style="width:110px" />
            <div class="filter-group">
              <el-select v-model="logFilters.action" :placeholder="$t('myLogs.actionPlaceholder')" clearable filterable multiple collapse-tags collapse-tags-tooltip size="small" style="width:160px">
                <el-option-group v-for="group in actionGroupsI18n" :key="group.label" :label="group.label">
                  <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
                </el-option-group>
              </el-select>
              <el-tooltip :content="actionExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
                <el-button size="small" :type="actionExcludeMode ? 'danger' : ''" @click="actionExcludeMode = !actionExcludeMode; loadLogs()" style="padding: 5px">
                  <el-icon :size="14"><component :is="actionExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <div class="filter-group">
              <el-select v-model="logFilters.target_type" :placeholder="$t('myLogs.targetTypePlaceholder')" clearable multiple collapse-tags collapse-tags-tooltip size="small" style="width:160px">
                <el-option v-for="item in targetTypeOptionsI18n" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-tooltip :content="targetTypeExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
                <el-button size="small" :type="targetTypeExcludeMode ? 'danger' : ''" @click="targetTypeExcludeMode = !targetTypeExcludeMode; loadLogs()" style="padding: 5px">
                  <el-icon :size="14"><component :is="targetTypeExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <el-date-picker v-model="logFilters.dateRange" type="daterange" :range-separator="$t('myLogs.dateRangeSeparator')" :start-placeholder="$t('myLogs.startDatePlaceholder')" :end-placeholder="$t('myLogs.endDatePlaceholder')" size="small" style="width:220px" value-format="YYYY-MM-DD" />
            <div class="filter-actions">
              <el-button size="small" type="primary" @click="loadLogs">{{ $t('common.search') }}</el-button>
              <el-button size="small" @click="resetLogFilters">{{ $t('common.reset') }}</el-button>
            </div>
          </div>
          <el-table :data="logs" stripe v-loading="loading" style="width:100%" @sort-change="handleLogSortChange">
            <el-table-column prop="id" :label="$t('admin.id')" width="60" />
            <el-table-column :label="$t('admin.user')" min-width="120">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="getSrc(row.user_id, row.user?.avatar)" :src="getSrc(row.user_id, row.user?.avatar)" :size="24" />
                  <el-avatar v-else :size="24">{{ firstLetter(row.user?.username) }}</el-avatar>
                  <span>{{ row.user?.username || 'User #' + row.user_id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="$t('admin.targetUserLabel')" min-width="120">
              <template #default="{ row }">
                <template v-if="row.target_user">
                  <div style="display:flex;align-items:center;gap:6px">
                    <el-avatar v-if="getSrc(row.target_user_id, row.target_user?.avatar)" :src="getSrc(row.target_user_id, row.target_user?.avatar)" :size="24" />
                    <el-avatar v-else :size="24">{{ firstLetter(row.target_user?.username) }}</el-avatar>
                    <span>{{ row.target_user.username }}</span>
                  </div>
                </template>
                <span v-else style="color:#909399">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="action" :label="$t('common.action')" min-width="120" sortable="custom" />
            <el-table-column prop="target_type" :label="$t('admin.targetTypeLabel')" min-width="100" sortable="custom" />
            <el-table-column prop="ip_address" :label="$t('common.ipAddress')" width="130" />
            <el-table-column :label="$t('common.detail')" min-width="200">
              <template #default="{ row }">
                <template v-if="row.detail && row.detail !== 'null'">
                  <el-tag v-for="(val, key) in parseDetail(row.detail)" :key="key" size="small" style="margin:2px" type="info">
                    {{ formatDetailKey(key) }}: {{ formatDetailValue(val) }}
                  </el-tag>
                </template>
                <span v-else style="color:#909399">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="$t('common.time')" width="170" sortable="custom" />
          </el-table>
          <el-pagination v-if="logTotal > 0" :current-page="logPage" :page-size="logPageSize" :total="logTotal"
            layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
            @current-change="handleLogPageChange" @size-change="handleLogSizeChange" style="margin-top:16px" />
        </el-tab-pane>
        <el-tab-pane :label="$t('admin.systemSettings')" name="settings">
          <el-form :model="smtpForm" label-width="100px" style="max-width:560px">
            <el-divider content-position="left">{{ $t('admin.smtpConfig') }}</el-divider>
            <el-form-item :label="$t('admin.smtpHost')">
              <el-input v-model="smtpForm.host" :placeholder="$t('admin.smtpHostPlaceholder')" />
            </el-form-item>
            <el-form-item :label="$t('admin.smtpPort')">
              <el-input-number v-model="smtpForm.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item :label="$t('admin.encryption')">
              <el-select v-model="smtpForm.tls_type" style="width:100%" @change="onTLSTypeChange">
                <el-option :label="$t('admin.tlsStarttls')" value="starttls" />
                <el-option :label="$t('admin.tlsImplicit')" value="implicit" />
                <el-option :label="$t('admin.tlsNone')" value="none" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('admin.loginEmail')">
              <el-input v-model="smtpForm.username" :placeholder="$t('admin.loginEmailPlaceholder')" autocomplete="off" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">{{ $t('admin.loginEmailTip') }}</div>
            </el-form-item>
            <el-form-item :label="$t('admin.authorizationCode')">
              <el-input v-model="smtpForm.password" type="password" show-password :placeholder="$t('admin.authCodePlaceholder')" autocomplete="new-password" />
            </el-form-item>
            <el-form-item :label="$t('admin.senderAddress')">
              <el-input v-model="smtpForm.from" :placeholder="$t('admin.senderAddressPlaceholder')" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">{{ $t('admin.senderAddressTip') }}</div>
            </el-form-item>
            <el-form-item :label="$t('admin.senderName')">
              <el-input v-model="smtpForm.from_name" placeholder="DomainNest" />
            </el-form-item>
            <el-form-item :label="$t('admin.testRecipient')">
              <el-input v-model="testEmail" :placeholder="$t('admin.testRecipientPlaceholder')" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveSMTP">{{ $t('admin.saveConfig') }}</el-button>
              <el-button @click="handleTestSMTP" :loading="testingSMTP" :disabled="!testEmail">{{ $t('admin.sendTestEmail') }}</el-button>
            </el-form-item>
          </el-form>
          <el-form :model="securityForm" label-width="160px" style="max-width:560px;margin-top:32px">
            <el-divider content-position="left">{{ $t('admin.securityConfig') }}</el-divider>
            <el-form-item :label="$t('admin.resetTokenExpiry')">
              <el-input-number v-model="securityForm.reset_token_expiry_minutes" :min="1" :max="60" />
              <span style="margin-left:8px;color:#909399;font-size:13px">{{ $t('admin.minutes') }}</span>
            </el-form-item>
            <el-form-item :label="$t('admin.minPasswordLength')">
              <el-input-number v-model="securityForm.min_password_length" :min="4" :max="32" />
              <span style="margin-left:8px;color:#909399;font-size:13px">{{ $t('admin.characters') }}</span>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveSecurity">{{ $t('admin.saveConfig') }}</el-button>
            </el-form-item>
          </el-form>
          <el-form :model="uploadsForm" label-width="160px" style="max-width:560px;margin-top:32px">
            <el-divider content-position="left">{{ $t('admin.uploadConfig') }}</el-divider>
            <el-form-item :label="$t('admin.maxAvatarSize')">
              <el-input-number v-model="uploadsForm.max_avatar_size_mb" :min="1" :max="20" />
              <span style="margin-left:8px;color:#909399;font-size:13px">{{ $t('admin.mb') }}</span>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveUploads">{{ $t('admin.saveConfig') }}</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane :label="$t('admin.providerManagement')" name="providers">
          <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap;align-items:center">
            <el-input v-model="providerSearch" :placeholder="$t('admin.providerSearchPlaceholder')" clearable style="width:200px" size="default" />
            <el-select v-model="providerTypeFilter" :placeholder="$t('admin.providerTypeFilter')" clearable style="width:150px" size="default">
              <el-option :label="$t('common.all')" value="" />
              <el-option v-for="pt in providerTypeOptions" :key="pt.value" :label="pt.label" :value="pt.value" />
            </el-select>
          </div>
          <el-table :data="filteredProviders" stripe v-loading="loadingAdminProviders" style="width:100%"
            @sort-change="handleProviderSortChange">
            <el-table-column prop="id" :label="$t('admin.id')" width="80" sortable="custom" />
            <el-table-column prop="name" :label="$t('admin.providerName')" min-width="120" sortable="custom" />
            <el-table-column prop="provider_type" :label="$t('admin.providerType')" width="120" />
            <el-table-column :label="$t('admin.belongsToUser')" min-width="120">
              <template #default="{ row }">
                {{ row.user?.username || row.user_id }}
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" sortable="custom" />
            <el-table-column :label="$t('common.action')" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openProviderDetail(row)">{{ $t('admin.viewDetail') }}</el-button>
                <el-button link type="success" size="small" @click="openEditProvider(row)">{{ $t('common.edit') }}</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteProvider(row)">{{ $t('common.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="$t('admin.inviteCodeManagement')" name="inviteCodes">
          <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap;align-items:center">
            <el-input v-model="inviteCodeFilter" :placeholder="$t('admin.filterByCreatorOrUsedBy')" clearable style="width:200px" size="default" />
            <el-select v-model="inviteCodeStatus" :placeholder="$t('common.all')" clearable style="width:130px" size="default">
              <el-option :label="$t('common.all')" value="all" />
              <el-option :label="$t('admin.codeUnused')" value="unused" />
              <el-option :label="$t('admin.codeUsed')" value="used" />
            </el-select>
          </div>
          <el-table :data="filteredInviteCodes" stripe v-loading="inviteCodeLoading" style="width:100%" @sort-change="handleInviteCodeSortChange">
            <el-table-column prop="code" :label="$t('admin.inviteCode')" min-width="120" sortable="custom">
              <template #header="{ column }">{{ $t('admin.inviteCode') }}</template>
            </el-table-column>
            <el-table-column :label="$t('admin.creator')" width="120">
              <template #default="{ row }">
                {{ row.creator?.username || row.creator_id }}
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" sortable="custom">
              <template #header="{ column }">{{ $t('common.createdAt') }}</template>
            </el-table-column>
            <el-table-column :label="$t('common.status')" width="100">
              <template #default="{ row }">
                <el-tag :type="row.used_by ? 'info' : 'success'" size="small">
                  {{ row.used_by ? $t('admin.codeUsed') : $t('admin.codeUnused') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="$t('admin.usedBy')" width="120">
              <template #default="{ row }">
                {{ row.used_by_user?.username || (row.used_by ? row.used_by : '-') }}
              </template>
            </el-table-column>
            <el-table-column prop="used_at" :label="$t('admin.usedAt')" width="170">
              <template #default="{ row }">
                {{ row.used_at || '-' }}
              </template>
            </el-table-column>
            <el-table-column :label="$t('common.action')" width="80" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" size="small" :disabled="!!row.used_by" @click="handleDeleteInviteCode(row)">{{ $t('common.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination v-if="inviteCodeTotal > 0" :current-page="inviteCodePage" :page-size="inviteCodePageSize" :total="inviteCodeTotal"
            layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
            @current-change="handleInviteCodePageChange" @size-change="handleInviteCodeSizeChange" style="margin-top:16px" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 编辑用户对话框 -->
    <el-dialog v-model="showEditUser" :title="$t('admin.editUser')" width="480px" destroy-on-close>
      <el-form :model="editForm" label-width="80px">
        <el-form-item :label="$t('common.username')">
          <el-input v-model="editForm.username" />
        </el-form-item>
        <el-form-item :label="$t('admin.role')">
          <el-select v-model="editForm.role" style="width:100%" :disabled="editTarget?.is_super_admin">
            <el-option :label="$t('admin.normalUser')" value="user" />
            <el-option :label="$t('common.admin')" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('common.status')">
          <el-select v-model="editForm.status" style="width:100%" :disabled="editTarget?.id === auth.user?.id">
            <el-option :label="$t('admin.statusNormal')" :value="1" />
            <el-option :label="$t('admin.statusDisabled')" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.inviteLimit')">
          <el-input :model-value="editForm.invite_limit" disabled />
        </el-form-item>
        <el-form-item :label="$t('admin.nickname')">
          <el-input v-model="editForm.nickname" />
        </el-form-item>
        <el-form-item :label="$t('admin.email')">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item :label="$t('admin.phone')">
          <el-input v-model="editForm.phone" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditUser = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleEditUser">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="showResetPwd" :title="$t('admin.resetPasswordTitle')" width="400px" destroy-on-close>
      <p style="margin-bottom:12px">{{ $t('admin.setNewPassword', { username: resetPwdTarget?.username }) }}</p>
      <el-input v-model="newPassword" type="password" :placeholder="$t('admin.newPasswordPlaceholder')" show-password />
      <template #footer>
        <el-button @click="showResetPwd = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleResetPwd">{{ $t('admin.confirmReset') }}</el-button>
      </template>
    </el-dialog>

    <!-- 单用户邀请额度操作对话框 -->
    <el-dialog v-model="showInviteAction" :title="inviteActionType === 'grant' ? $t('admin.grantInviteTitle') : $t('admin.revokeInviteTitle')" width="400px" destroy-on-close>
      <p>{{ $t('admin.inviteActionTarget', { username: inviteTarget?.username }) }}</p>
      <el-form label-width="80px">
        <el-form-item :label="$t('admin.inviteAmount')">
          <el-input-number v-model="inviteActionAmount" :min="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showInviteAction = false">{{ $t('common.cancel') }}</el-button>
        <el-button :type="inviteActionType === 'grant' ? 'primary' : 'danger'" :loading="inviteActionLoading" @click="handleInviteAction">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 批量邀请额度操作对话框 -->
    <el-dialog v-model="showBatchInvite" :title="batchInviteType === 'grant' ? $t('admin.batchGrantTitle') : $t('admin.batchRevokeTitle')" width="480px" destroy-on-close>
      <p style="margin-bottom:12px">{{ $t('admin.selectedCount', { count: selectedUsers.length }) }}: {{ selectedUsers.map(u => u.username).join(', ') }}</p>
      <p v-if="batchInviteType === 'grant'" style="color:#909399;font-size:13px;margin-bottom:12px">{{ $t('admin.batchGrantHint') }}</p>
      <p v-else style="color:#909399;font-size:13px;margin-bottom:12px">{{ $t('admin.batchRevokeHint') }}</p>
      <el-form label-width="80px">
        <el-form-item :label="$t('admin.inviteAmount')">
          <el-input-number v-model="batchInviteAmount" :min="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBatchInvite = false">{{ $t('common.cancel') }}</el-button>
        <el-button :type="batchInviteType === 'grant' ? 'primary' : 'danger'" :loading="batchInviteLoading" @click="handleBatchInvite">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 提供商详情对话框 -->
    <el-dialog v-model="showProviderDetail" :title="$t('admin.providerDetailTitle')" width="600px" destroy-on-close>
      <div v-if="providerDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('admin.id')">{{ providerDetail.id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('admin.providerName')">{{ providerDetail.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('admin.providerType')">{{ providerDetail.provider_type }}</el-descriptions-item>
          <el-descriptions-item :label="$t('admin.providerEndpoint')">{{ providerDetail.endpoint || $t('common.default') }}</el-descriptions-item>
          <el-descriptions-item :label="$t('admin.belongsToUser')">{{ providerDetail.user?.username || providerDetail.user_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.createdAt')">{{ providerDetail.created_at }}</el-descriptions-item>
        </el-descriptions>
        <el-divider>{{ $t('admin.providerDomainsCount', { count: providerDomains.length }) }}</el-divider>
        <el-table :data="providerDomains" stripe size="small" max-height="300">
          <el-table-column prop="domain_name" :label="$t('domainDetail.fullDomain')" min-width="160" show-overflow-tooltip />
          <el-table-column prop="record_count" :label="$t('providers.recordCount')" width="80" />
          <el-table-column :label="$t('admin.claimStatus')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.claimed ? 'success' : 'info'" size="small">
                {{ row.claimed ? $t('admin.claimed') : $t('admin.unclaimed') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('admin.owner')" width="100">
            <template #default="{ row }">
              {{ row.owner_name || '-' }}
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showProviderDetail = false">{{ $t('common.back') }}</el-button>
      </template>
    </el-dialog>

    <!-- 编辑提供商对话框 -->
    <el-dialog v-model="showEditProvider" :title="$t('admin.editProvider')" width="480px" destroy-on-close>
      <el-form :model="editProviderForm" label-width="100px">
        <el-form-item :label="$t('admin.providerName')">
          <el-input v-model="editProviderForm.name" />
        </el-form-item>
        <el-form-item :label="$t('admin.providerEndpoint')">
          <el-input v-model="editProviderForm.endpoint" :placeholder="$t('admin.providerEndpoint')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditProvider = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="editProviderLoading" @click="handleUpdateProvider">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listUsers, listLogs, createRootDomain, updateUser, adminResetPassword, disableUser, getSettings, updateSettings, testSMTP, promoteToAdmin, demoteFromAdmin, getAdminDomainTree, adminBatchDeleteDomains, listInviteCodes, deleteInviteCode, adminListProviders, getAdminProviderDetail, adminUpdateProvider, adminDeleteProvider } from '../api/admin'
import { listProviders, listProviderDomains } from '../api/provider'
import { grantInviteQuota, revokeInviteQuota } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { actionGroups, targetTypeOptions } from '../constants/operationLogs'
import { useAvatar } from '../composables/useAvatar'
import { useError } from '../composables/useError'

const auth = useAuthStore()
const route = useRoute()
const { t } = useI18n()
const { getSrc, firstLetter } = useAvatar()
const { showError } = useError()

const activeTab = ref(route.query.tab || 'users')
const users = ref([])
const logs = ref([])
const domainTree = ref([])
const domainTreeSearch = ref('')
const loading = ref(false)

const adminProviders = ref([])
const loadingAdminProviders = ref(false)
const adminDomains = ref([])
const selectedProviderId = ref(null)
const selectedDomain = ref('')
const loadingAdminDomains = ref(false)

const inviteCodes = ref([])
const inviteCodeLoading = ref(false)
const inviteCodePage = ref(1)
const inviteCodePageSize = ref(20)
const inviteCodeTotal = ref(0)
const inviteCodeFilter = ref('')
const inviteCodeStatus = ref('')
const inviteCodeSortBy = ref('created_at')
const inviteCodeSortOrder = ref('desc')

const providerSearch = ref('')
const providerTypeFilter = ref('')
const providerTypes = [
  { value: 'aliyun', label: 'Alibaba Cloud DNS' },
  { value: 'aliesa', label: 'Alibaba Cloud ESA' },
  { value: 'tencentcloud', label: 'Tencent Cloud DNSPod' },
  { value: 'cloudflare', label: 'Cloudflare' },
  { value: 'huaweicloud', label: 'Huawei Cloud DNS' },
  { value: 'baiducloud', label: 'Baidu Cloud DNS' },
  { value: 'trafficroute', label: 'Volcengine DNS' },
  { value: 'godaddy', label: 'GoDaddy' },
  { value: 'namecheap', label: 'Namecheap' },
  { value: 'namesilo', label: 'NameSilo' },
  { value: 'porkbun', label: 'Porkbun' },
  { value: 'vercel', label: 'Vercel' },
  { value: 'gcore', label: 'Gcore' },
  { value: 'nsone', label: 'NS1' },
  { value: 'name_com', label: 'Name.com' },
  { value: 'cloudns', label: 'ClouDNS' },
  { value: 'dnsla', label: 'DNSLA' },
  { value: 'dynadot', label: 'Dynadot' },
  { value: 'dynv6', label: 'dynv6' },
  { value: 'spaceship', label: 'Spaceship' },
  { value: 'edgeone', label: 'EdgeOne' },
  { value: 'rainyun', label: 'Rain Yun' },
  { value: 'hipmdnsmgr', label: 'HiPM DNS' },
  { value: 'nowcn', label: 'Nowcn' },
  { value: 'eranet', label: 'Eranet' },
  { value: 'tnethk', label: 'Tnethk' },
]
const providerTypeLabelKeys = {
  aliyun: 'providers.providerAliyun',
  aliesa: 'providers.providerAliesa',
  tencentcloud: 'providers.providerTencentcloud',
  cloudflare: 'providers.providerCloudflare',
  huaweicloud: 'providers.providerHuaweicloud',
  baiducloud: 'providers.providerBaiducloud',
  trafficroute: 'providers.providerTrafficroute',
  godaddy: 'providers.providerGodaddy',
  namecheap: 'providers.providerNamecheap',
  namesilo: 'providers.providerNamesilo',
  porkbun: 'providers.providerPorkbun',
  vercel: 'providers.providerVercel',
  gcore: 'providers.providerGcore',
  nsone: 'providers.providerNsone',
  name_com: 'providers.providerName_com',
  cloudns: 'providers.providerCloudns',
  dnsla: 'providers.providerDnsla',
  dynadot: 'providers.providerDynadot',
  dynv6: 'providers.providerDynv6',
  spaceship: 'providers.providerSpaceship',
  edgeone: 'providers.providerEdgeone',
  rainyun: 'providers.providerRainyun',
  hipmdnsmgr: 'providers.providerHipmdnsmgr',
  nowcn: 'providers.providerNowcn',
  eranet: 'providers.providerEranet',
  tnethk: 'providers.providerTnethk',
}
const providerTypeOptions = computed(() =>
  providerTypes.map(p => ({
    ...p,
    label: providerTypeLabelKeys[p.value] ? t(providerTypeLabelKeys[p.value]) : p.label,
  }))
)
const providerSortBy = ref('id')
const providerSortOrder = ref('desc')
const showProviderDetail = ref(false)
const providerDetail = ref(null)
const providerDomains = ref([])
const showEditProvider = ref(false)
const editProviderForm = reactive({ name: '', endpoint: '' })
const editProviderLoading = ref(false)
const editProviderTarget = ref(null)

const showEditUser = ref(false)
const editTarget = ref(null)
const editForm = reactive({ username: '', role: '', status: 1, invite_limit: 5, nickname: '', email: '', phone: '' })

const showResetPwd = ref(false)
const resetPwdTarget = ref(null)
const newPassword = ref('')

const selectedUsers = ref([])
const selectedDomains = ref([])
const batchDeleteLoading = ref(false)
const showInviteAction = ref(false)
const inviteActionType = ref('grant')
const inviteTarget = ref(null)
const inviteActionAmount = ref(1)
const inviteActionLoading = ref(false)
const showBatchInvite = ref(false)
const batchInviteType = ref('grant')
const batchInviteAmount = ref(1)
const batchInviteLoading = ref(false)

const smtpForm = reactive({ host: '', port: 587, username: '', password: '', from: '', from_name: 'DomainNest', tls_type: 'starttls' })
const testingSMTP = ref(false)
const testEmail = ref('')

const securityForm = reactive({ reset_token_expiry_minutes: 5, min_password_length: 6 })
const uploadsForm = reactive({ max_avatar_size_mb: 2 })

const logFilters = reactive({ user_id: '', target_user_id: '', action: [], target_type: [], q: '', q_exclude: '', dateRange: null })
const actionExcludeMode = ref(false)
const targetTypeExcludeMode = ref(false)
const logSortBy = ref('created_at')
const logSortOrder = ref('desc')

const DETAIL_KEY_I18N = {
  username: 'common.username',
  full_domain: 'admin.detailKeyFullDomain',
  host: 'admin.detailKeyHost',
  type: 'admin.detailKeyType',
  value: 'admin.detailKeyValue',
  amount: 'admin.detailKeyAmount',
  enabled: 'admin.detailKeyStatus',
  ids: 'admin.detailKeyIds',
  target_user_id: 'admin.detailKeyTargetUser',
  invited_by: 'admin.detailKeyInvitedBy',
  provider_name: 'admin.detailKeyProvider',
}

const parseDetail = (json) => {
  try { return JSON.parse(json) } catch { return {} }
}

const formatDetailKey = (key) => {
  const i18nKey = DETAIL_KEY_I18N[key]
  return i18nKey ? t(i18nKey) : key
}

const formatDetailValue = (val) => {
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'boolean') return t(val ? 'common.enabled' : 'common.disabled')
  return String(val)
}

const getDomainLabel = (d) => d.domain_name + (d.record_count != null ? ` (${d.record_count} ${t('admin.recordCount')})` : '')

const GROUP_I18N_KEYS = {
  '域名': 'admin.actionGroupDomain',
  '记录': 'admin.actionGroupRecord',
  '权限': 'admin.actionGroupPermission',
  '用户': 'admin.actionGroupUser',
  '邀请': 'admin.actionGroupInvite',
  '好友': 'admin.actionGroupFriend',
  '提供商': 'admin.actionGroupProvider',
  '设置': 'admin.actionGroupSetting',
}

const ACTION_I18N_KEYS = {
  create_domain: 'admin.actionCreateDomain',
  transfer_domain: 'admin.actionTransferDomain',
  delete_domain: 'admin.actionDeleteDomain',
  create_root_domain: 'admin.actionCreateRootDomain',
  assign_domain: 'admin.actionAssignDomain',
  create_record: 'admin.actionCreateRecord',
  update_record: 'admin.actionUpdateRecord',
  delete_record: 'admin.actionDeleteRecord',
  toggle_record: 'admin.actionToggleRecord',
  batch_delete: 'admin.actionBatchDelete',
  batch_toggle: 'admin.actionBatchToggle',
  import: 'admin.actionImport',
  grant_permission: 'admin.actionGrantPermission',
  revoke_permission: 'admin.actionRevokePermission',
  revoke_request: 'admin.actionRevokeRequest',
  accept_return: 'admin.actionAcceptReturn',
  reject_return: 'admin.actionRejectReturn',
  assign_pending_records: 'admin.actionAssignPendingRecords',
  delete_pending_records: 'admin.actionDeletePendingRecords',
  register: 'admin.actionRegister',
  login: 'admin.actionLogin',
  update_profile: 'admin.actionUpdateProfile',
  change_password: 'admin.actionChangePassword',
  reset_token: 'admin.actionResetToken',
  upload_avatar: 'admin.actionUploadAvatar',
  delete_account: 'admin.actionDeleteAccount',
  update_user: 'admin.actionUpdateUser',
  admin_reset_password: 'admin.actionAdminResetPassword',
  disable_user: 'admin.actionDisableUser',
  promote_to_admin: 'admin.actionPromoteToAdmin',
  demote_from_admin: 'admin.actionDemoteFromAdmin',
  grant_invite: 'admin.actionGrantInvite',
  revoke_invite: 'admin.actionRevokeInvite',
  permission_granted: 'admin.actionPermissionGranted',
  permission_revoked: 'admin.actionPermissionRevoked',
  invite_granted: 'admin.actionInviteGranted',
  invite_revoked: 'admin.actionInviteRevoked',
  admin_grant: 'admin.actionAdminGrant',
  send_friend_request: 'admin.actionSendFriendRequest',
  accept_friend: 'admin.actionAcceptFriend',
  reject_friend: 'admin.actionRejectFriend',
  remove_friend: 'admin.actionRemoveFriend',
  create_provider: 'admin.actionCreateProvider',
  update_provider: 'admin.actionUpdateProvider',
  delete_provider: 'admin.actionDeleteProvider',
  claim_domain: 'admin.actionClaimDomain',
  update_settings: 'admin.actionUpdateSettings',
  smtp_test: 'admin.actionSmtpTest',
}

const TARGET_I18N_KEYS = {
  domain_node: 'admin.targetDomainNode',
  dns_record: 'admin.targetDnsRecord',
  user: 'admin.targetUser',
  setting: 'admin.targetSetting',
  provider: 'admin.targetProvider',
  friend: 'admin.targetFriend',
}

const actionGroupsI18n = computed(() => {
  return actionGroups.map(group => ({
    ...group,
    label: t(GROUP_I18N_KEYS[group.label] || group.label),
    options: group.options.map(opt => ({
      ...opt,
      label: t(ACTION_I18N_KEYS[opt.value] || opt.label)
    }))
  }))
})

const targetTypeOptionsI18n = computed(() => {
  return targetTypeOptions.map(opt => ({
    ...opt,
    label: t(TARGET_I18N_KEYS[opt.value] || opt.label)
  }))
})
const logPage = ref(1)
const logPageSize = ref(20)
const logTotal = ref(0)


const loadData = async () => {
  loading.value = true
  try {
    const [usersRes] = await Promise.all([
      listUsers(),
    ])
    users.value = usersRes.data
    await loadLogs()
    await loadDomainTree()
  } catch {
    // silently ignore — shows empty state
  } finally {
    loading.value = false
  }
  loadSMTP()
  loadSecurity()
  loadUploads()
  loadAdminProviders()
  loadInviteCodes()
}

const loadDomainTree = async () => {
  try {
    const res = await getAdminDomainTree()
    domainTree.value = res.data || []
  } catch { /* ignore */ }
}

const filteredDomainTree = computed(() => {
  if (!domainTreeSearch.value) return domainTree.value
  const q = domainTreeSearch.value.toLowerCase()
  const filterNode = (nodes) => {
    const result = []
    for (const node of nodes) {
      const children = node.children ? filterNode(node.children) : []
      if (node.full_domain?.toLowerCase().includes(q) || children.length > 0) {
        result.push({ ...node, children })
      }
    }
    return result
  }
  return filterNode(domainTree.value)
})

const filteredInviteCodes = computed(() => {
  let result = [...inviteCodes.value]
  const q = inviteCodeFilter.value.toLowerCase()
  if (q) {
    result = result.filter(c =>
      (c.creator?.username || '').toLowerCase().includes(q) ||
      (c.used_by_user?.username || c.used_by || '').toLowerCase().includes(q)
    )
  }
  if (inviteCodeStatus.value === 'unused') {
    result = result.filter(c => !c.used_by)
  } else if (inviteCodeStatus.value === 'used') {
    result = result.filter(c => !!c.used_by)
  }
  result.sort((a, b) => {
    let aVal = a[inviteCodeSortBy.value]
    let bVal = b[inviteCodeSortBy.value]
    if (aVal == null) aVal = ''
    if (bVal == null) bVal = ''
    if (typeof aVal === 'string') aVal = aVal.toLowerCase()
    if (typeof bVal === 'string') bVal = bVal.toLowerCase()
    if (aVal < bVal) return inviteCodeSortOrder.value === 'asc' ? -1 : 1
    if (aVal > bVal) return inviteCodeSortOrder.value === 'asc' ? 1 : -1
    return 0
  })
  return result
})

const filteredProviders = computed(() => {
  let result = [...adminProviders.value]
  const q = providerSearch.value.toLowerCase()
  if (q) {
    result = result.filter(p =>
      (p.name || '').toLowerCase().includes(q) ||
      (p.provider_type || '').toLowerCase().includes(q)
    )
  }
  if (providerTypeFilter.value) {
    result = result.filter(p => p.provider_type === providerTypeFilter.value)
  }
  result.sort((a, b) => {
    let aVal = a[providerSortBy.value]
    let bVal = b[providerSortBy.value]
    if (aVal == null) aVal = ''
    if (bVal == null) bVal = ''
    if (typeof aVal === 'string') aVal = aVal.toLowerCase()
    if (typeof bVal === 'string') bVal = bVal.toLowerCase()
    if (aVal < bVal) return providerSortOrder.value === 'asc' ? -1 : 1
    if (aVal > bVal) return providerSortOrder.value === 'asc' ? 1 : -1
    return 0
  })
  return result
})

const loadAdminProviders = async () => {
  loadingAdminProviders.value = true
  try {
    const res = await adminListProviders()
    adminProviders.value = res.data || []
  } catch { /* ignore */ } finally {
    loadingAdminProviders.value = false
  }
}

watch(selectedProviderId, async (id) => {
  adminDomains.value = []
  selectedDomain.value = ''
  if (!id) return
  loadingAdminDomains.value = true
  try {
    const res = await listProviderDomains(id, { skipErrorToast: true })
    adminDomains.value = res.data || []
  } catch (e) {
    showError(e.response?.data?.message || t('admin.fetchDomainsFailed'))
  } finally {
    loadingAdminDomains.value = false
  }
})

const loadLogs = async () => {
  const params = { page: logPage.value, page_size: logPageSize.value, sort_by: logSortBy.value, sort_order: logSortOrder.value }
  if (logFilters.user_id) params.user_id = logFilters.user_id
  if (logFilters.target_user_id) params.target_user_id = logFilters.target_user_id
  if (logFilters.action.length > 0) {
    if (actionExcludeMode.value) {
      params.action_exclude = logFilters.action.join(',')
    } else {
      params.action = logFilters.action.join(',')
    }
  }
  if (logFilters.target_type.length > 0) {
    if (targetTypeExcludeMode.value) {
      params.target_type_exclude = logFilters.target_type.join(',')
    } else {
      params.target_type = logFilters.target_type.join(',')
    }
  }
  if (logFilters.q) params.q = logFilters.q
  if (logFilters.q_exclude) params.q_exclude = logFilters.q_exclude
  if (logFilters.dateRange && logFilters.dateRange.length === 2) {
    params.start_time = logFilters.dateRange[0]
    params.end_time = logFilters.dateRange[1]
  }
  const logsRes = await listLogs(params)
  logs.value = logsRes.data.items
  logTotal.value = logsRes.data.total
}

const resetLogFilters = () => {
  logFilters.user_id = ''
  logFilters.target_user_id = ''
  logFilters.action = []
  logFilters.target_type = []
  logFilters.q = ''
  logFilters.q_exclude = ''
  logFilters.dateRange = null
  actionExcludeMode.value = false
  targetTypeExcludeMode.value = false
  logPage.value = 1
  loadLogs()
}

const handleLogPageChange = (p) => {
  logPage.value = p
  loadLogs()
}

const handleLogSizeChange = (s) => {
  logPageSize.value = s
  logPage.value = 1
  loadLogs()
}

const handleLogSortChange = ({ prop, order }) => {
  logSortBy.value = prop || 'created_at'
  logSortOrder.value = order === 'ascending' ? 'asc' : 'desc'
  loadLogs()
}

const loadSMTP = async () => {
  try {
    const res = await getSettings('smtp')
    if (res.data) {
      Object.assign(smtpForm, res.data)
      if (res.data.test_email) {
        testEmail.value = res.data.test_email
      } else if (!testEmail.value && smtpForm.username) {
        testEmail.value = smtpForm.username
      }
    }
  } catch { /* no settings yet */ }
}

const loadSecurity = async () => {
  try {
    const res = await getSettings('security')
    if (res.data) {
      Object.assign(securityForm, res.data)
    }
  } catch { /* no settings yet */ }
}

const handleSaveSecurity = async () => {
  await updateSettings('security', { ...securityForm })
  ElMessage.success(t('admin.securityConfigSaved'))
}

const loadUploads = async () => {
  try {
    const res = await getSettings('uploads')
    if (res.data) {
      Object.assign(uploadsForm, res.data)
    }
  } catch { /* no settings yet */ }
}

const handleSaveUploads = async () => {
  await updateSettings('uploads', { ...uploadsForm })
  ElMessage.success(t('admin.uploadConfigSaved'))
}

const handleCreateRoot = async () => {
  if (adminProviders.length === 0) return
  if (!selectedProviderId.value || !selectedDomain.value) {
    ElMessage.warning(t('admin.selectProviderAndDomain'))
    return
  }
  try {
    await createRootDomain({ provider_id: selectedProviderId.value, domain_name: selectedDomain.value })
    ElMessage.success(t('admin.rootDomainCreated'))
    selectedDomain.value = ''
    loadData()
  } catch { /* error shown by interceptor */ }
}

const openEditUser = (row) => {
  editTarget.value = row
  editForm.username = row.username || ''
  editForm.role = row.role
  editForm.status = row.status
  editForm.invite_limit = row.invite_limit ?? 5
  editForm.nickname = row.nickname || ''
  editForm.email = row.email || ''
  editForm.phone = row.phone || ''
  showEditUser.value = true
}


const handleEditUser = async () => {
  const target = editTarget.value
  // Role changes must go through dedicated promote/demote endpoints (super_admin guarded)
  if (editForm.role !== target.role) {
    if (editForm.role === 'admin') {
      await promoteToAdmin(target.id)
    } else {
      await demoteFromAdmin(target.id)
    }
  }
  // Other fields via updateUser (excluding role)
  const { role, ...rest } = editForm
  await updateUser(target.id, rest)
  ElMessage.success(t('admin.userUpdated'))
  showEditUser.value = false
  loadData()
}

const openResetPwd = (row) => {
  resetPwdTarget.value = row
  newPassword.value = ''
  showResetPwd.value = true
}

const handleResetPwd = async () => {
  if (!newPassword.value || newPassword.value.length < 6) {
    ElMessage.warning(t('admin.passwordMinLength'))
    return
  }
  await adminResetPassword(resetPwdTarget.value.id, { new_password: newPassword.value })
  ElMessage.success(t('admin.passwordReset'))
  showResetPwd.value = false
}

const handleSelectionChange = (selection) => { selectedUsers.value = selection }

const handleDomainSelectionChange = (selection) => { selectedDomains.value = selection }

const handleBatchDeleteDomains = async () => {
  try {
    await ElMessageBox.confirm(
      t('admin.batchDeleteConfirmText', { count: selectedDomains.value.length }),
      t('common.confirmDelete'),
      { type: 'warning' }
    )
    batchDeleteLoading.value = true
    const ids = selectedDomains.value.map(d => d.id)
    const res = await adminBatchDeleteDomains({ node_ids: ids })
    ElMessage.success(res.message || t('admin.batchDeleteSuccess'))
    selectedDomains.value = []
    await loadDomainTree()
  } catch (e) {
    if (e === 'cancel') return
    showError(e.response?.data?.message || t('admin.batchDeleteFailed'))
  } finally {
    batchDeleteLoading.value = false
  }
}

const openGrantDialog = (row) => {
  inviteTarget.value = row
  inviteActionType.value = 'grant'
  inviteActionAmount.value = 1
  showInviteAction.value = true
}

const openRevokeDialog = (row) => {
  inviteTarget.value = row
  inviteActionType.value = 'revoke'
  inviteActionAmount.value = 1
  showInviteAction.value = true
}

const handleInviteAction = async () => {
  inviteActionLoading.value = true
  try {
    const data = { target_user_id: inviteTarget.value.id, amount: inviteActionAmount.value }
    if (inviteActionType.value === 'grant') {
      await grantInviteQuota(data, { skipErrorToast: true })
    } else {
      await revokeInviteQuota(data, { skipErrorToast: true })
    }
    ElMessage.success(inviteActionType.value === 'grant' ? t('profile.grantSuccess') : t('profile.revokeSuccess'))
    showInviteAction.value = false
    loadData()
  } catch (e) {
    showError(e.response?.data?.message || t('admin.requestFailed'))
  } finally {
    inviteActionLoading.value = false
  }
}

const openBatchGrant = () => { batchInviteType.value = 'grant'; batchInviteAmount.value = 1; showBatchInvite.value = true }
const openBatchRevoke = () => { batchInviteType.value = 'revoke'; batchInviteAmount.value = 1; showBatchInvite.value = true }

const handleBatchInvite = async () => {
  batchInviteLoading.value = true
  try {
    const promises = selectedUsers.value.map(u => {
      const data = { target_user_id: u.id, amount: batchInviteAmount.value }
      return batchInviteType.value === 'grant' ? grantInviteQuota(data, { skipErrorToast: true }) : revokeInviteQuota(data, { skipErrorToast: true })
    })
    await Promise.all(promises)
    ElMessage.success(batchInviteType.value === 'grant' ? t('admin.batchGrantSuccess') : t('admin.batchRevokeSuccess'))
    showBatchInvite.value = false
    selectedUsers.value = []
    loadData()
  } catch (e) {
    showError(e.response?.data?.message || t('admin.requestFailed'))
  } finally {
    batchInviteLoading.value = false
  }
}

const handleDisable = async (row) => {
  await ElMessageBox.confirm(t('admin.confirmDisableUser', { username: row.username }), t('common.confirm'), { type: 'warning' })
  await disableUser(row.id)
  ElMessage.success(t('admin.userDisabled'))
  loadData()
}

const handleEnable = async (row) => {
  await updateUser(row.id, { status: 1 })
  ElMessage.success(t('admin.userEnabled'))
  loadData()
}

const handleSaveSMTP = async () => {
  await updateSettings('smtp', { ...smtpForm, test_email: testEmail.value })
  ElMessage.success(t('admin.smtpConfigSaved'))
}

const portMap = { starttls: 587, implicit: 465, none: 25 }
const onTLSTypeChange = (val) => {
  if (portMap[val]) smtpForm.port = portMap[val]
}

const handleTestSMTP = async () => {
  testingSMTP.value = true
  try {
    await testSMTP({ to: testEmail.value }, { skipErrorToast: true })
    ElMessage.success(t('admin.testEmailSent'))
  } catch (e) {
    showError(t('admin.sendFailed') + (e.response?.data?.message || e.message))
  } finally {
    testingSMTP.value = false
  }
}

const loadInviteCodes = async () => {
  inviteCodeLoading.value = true
  try {
    const res = await listInviteCodes({ page: inviteCodePage.value, page_size: inviteCodePageSize.value })
    inviteCodes.value = res.data.items || []
    inviteCodeTotal.value = res.data.total || 0
  } catch { /* ignore */ } finally {
    inviteCodeLoading.value = false
  }
}

const handleDeleteInviteCode = async (row) => {
  try {
    await ElMessageBox.confirm(t('admin.confirmDeleteInviteCode'), t('common.confirmDelete'), { type: 'warning' })
    await deleteInviteCode(row.id)
    ElMessage.success(t('admin.inviteCodeDeleted'))
    loadInviteCodes()
  } catch { /* cancelled or error */ }
}

const handleInviteCodePageChange = (p) => {
  inviteCodePage.value = p
  loadInviteCodes()
}

const handleInviteCodeSizeChange = (s) => {
  inviteCodePageSize.value = s
  inviteCodePage.value = 1
  loadInviteCodes()
}

const handleInviteCodeSortChange = ({ prop, order }) => {
  inviteCodeSortBy.value = prop || 'created_at'
  inviteCodeSortOrder.value = order === 'ascending' ? 'asc' : 'desc'
}

const openProviderDetail = async (row) => {
  try {
    const res = await getAdminProviderDetail(row.id)
    providerDetail.value = res.data.provider
    providerDomains.value = res.data.domains || []
    showProviderDetail.value = true
  } catch (e) {
    showError(e.response?.data?.message || t('admin.requestFailed'))
  }
}

const openEditProvider = (row) => {
  editProviderTarget.value = row
  editProviderForm.name = row.name || ''
  editProviderForm.endpoint = row.endpoint || ''
  showEditProvider.value = true
}

const handleUpdateProvider = async () => {
  editProviderLoading.value = true
  try {
    await adminUpdateProvider(editProviderTarget.value.id, { name: editProviderForm.name, endpoint: editProviderForm.endpoint })
    ElMessage.success(t('admin.providerUpdated'))
    showEditProvider.value = false
    loadAdminProviders()
  } catch (e) {
    showError(e.response?.data?.message || t('admin.requestFailed'))
  } finally {
    editProviderLoading.value = false
  }
}

const handleDeleteProvider = async (row) => {
  try {
    await ElMessageBox.confirm(
      t('admin.deleteProviderConfirm', { name: row.name, count: 0 }),
      t('common.confirmDelete'),
      { type: 'warning' }
    )
    await adminDeleteProvider(row.id, true)
    ElMessage.success(t('admin.providerDeleted'))
    loadAdminProviders()
  } catch (e) {
    if (e === 'cancel') return
    const errMsg = e.response?.data?.message || ''
    if (errMsg.includes('归档')) {
      try {
        await ElMessageBox.confirm(errMsg, t('common.confirmDelete'), { type: 'warning' })
        await adminDeleteProvider(row.id, true)
        ElMessage.success(t('admin.providerDeleted'))
        loadAdminProviders()
      } catch { /* cancelled */ }
    } else {
      showError(e.response?.data?.message || t('admin.requestFailed'))
    }
  }
}

const handleProviderSortChange = ({ prop, order }) => {
  providerSortBy.value = prop || 'id'
  providerSortOrder.value = order === 'ascending' ? 'asc' : 'desc'
}


onMounted(loadData)
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
.domain-actions {
  margin-bottom: 16px;
}
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.filter-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.filter-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.domain-tree-layout {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .domain-tree-layout {
    flex-direction: column;
  }
  .domain-actions :deep(.el-form--inline .el-form-item) {
    display: block;
    margin-right: 0;
    margin-bottom: 12px;
  }
  .domain-actions :deep(.el-form--inline) {
    display: block;
  }
  :deep(.el-table) {
    font-size: 12px;
  }
  :deep(.el-table .cell) {
    padding: 0 6px;
  }
  /* Hide less critical columns on mobile */
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
  }
  :deep(.el-tabs__item) {
    padding: 0 10px;
    font-size: 13px;
  }
}
</style>
