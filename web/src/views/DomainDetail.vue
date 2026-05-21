<!-- web/src/views/DomainDetail.vue -->
<template>
  <div>
    <el-button text @click="$router.push('/dashboard')" style="margin-bottom:12px">
      <el-icon><component :is="'ArrowLeft'" /></el-icon>
      {{ $t('common.back') }}
    </el-button>

    <!-- Domain info bar -->
    <el-card v-if="domain" class="domain-info-bar" shadow="never">
      <div class="info-bar-content">
        <div class="info-bar-left">
          <span class="info-bar-domain">{{ domain.full_domain }}</span>
          <el-tag size="small" type="info">#{{ domain.id }}</el-tag>
          <el-tag v-if="domain.is_materialized" size="small" type="success">{{ $t('domainDetail.materialized') }}</el-tag>
          <span class="info-bar-meta">{{ $t('common.createdAt') }}: {{ domain.created_at }}</span>
        </div>
        <div class="info-bar-right" v-if="domain.owner_id === auth.user?.id">
          <el-button size="small" type="warning" @click="showTransfer = true">{{ $t('domainDetail.transferDomain') }}</el-button>
          <el-button size="small" type="danger" @click="handleDeleteDomain">{{ $t('domainDetail.deleteDomain') }}</el-button>
          <el-button v-if="domain.is_materialized" size="small" type="warning" plain @click="handleDemoteNode">{{ $t('domainDetail.demoteNode') }}</el-button>
        </div>
      </div>
    </el-card>

    <!-- Tabbed content -->
    <el-tabs v-model="activeTab" class="domain-detail-tabs">
      <!-- DNS Records Tab -->
      <el-tab-pane :label="$t('domainDetail.tabRecords')" name="records">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>{{ $t('domainDetail.dnsRecords') }}</span>
              <div class="header-actions">
                <el-button size="small" @click="handleExport('json')">{{ $t('domainDetail.exportJson') }}</el-button>
                <el-button size="small" @click="handleExport('csv')">{{ $t('domainDetail.exportCsv') }}</el-button>
                <el-button size="small" @click="showImport = true">{{ $t('domainDetail.import') }}</el-button>
                <el-button type="primary" size="small" @click="openAddRecord">
                  <el-icon><component :is="'Plus'" /></el-icon>
                  {{ $t('domainDetail.addRecord') }}
                </el-button>
              </div>
            </div>
          </template>

          <!-- search / filter bar -->
          <div class="filter-bar">
            <el-input v-model="filters.host" :placeholder="$t('domainDetail.host')" clearable size="small" style="width:140px" @clear="loadRecords" @keyup.enter="loadRecords" />
            <el-select v-model="filters.record_type" :placeholder="$t('domainDetail.recordType')" clearable size="small" style="width:120px" @change="loadRecords">
              <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
            </el-select>
            <el-input v-model="filters.value" :placeholder="$t('domainDetail.value')" clearable size="small" style="width:160px" @clear="loadRecords" @keyup.enter="loadRecords" />
            <el-select v-model="filters.enabled" :placeholder="$t('domainDetail.enabledStatus')" clearable size="small" style="width:100px" @change="loadRecords">
              <el-option :label="$t('common.enabled')" :value="true" />
              <el-option :label="$t('common.disabled')" :value="false" />
            </el-select>
            <el-select v-model="filters.sync_status" :placeholder="$t('domainDetail.syncStatus')" clearable size="small" style="width:110px" @change="loadRecords">
              <el-option :label="$t('common.synced')" value="synced" />
              <el-option :label="$t('domainDetail.pendingSync')" value="pending" />
              <el-option :label="$t('domainDetail.syncFailed')" value="failed" />
              <el-option :label="$t('domainDetail.disabled')" value="disabled" />
            </el-select>
            <el-button size="small" type="primary" @click="loadRecords">{{ $t('common.search') }}</el-button>
          </div>

          <!-- batch action bar -->
          <div class="batch-bar" v-if="selectedIds.length > 0">
            <span>{{ $t('domainDetail.selectedCount', { count: selectedIds.length }) }}</span>
            <el-button size="small" type="success" @click="handleBatchToggle(true)">{{ $t('domainDetail.batchEnable') }}</el-button>
            <el-button size="small" type="warning" @click="handleBatchToggle(false)">{{ $t('domainDetail.batchDisable') }}</el-button>
            <el-button size="small" type="danger" @click="handleBatchDelete">{{ $t('domainDetail.batchDelete') }}</el-button>
            <el-button size="small" type="warning" @click="showBatchTransfer = true">{{ $t('domainDetail.batchTransfer') }}</el-button>
          </div>

          <el-table :data="records" stripe v-loading="loading" @selection-change="handleSelectionChange" row-key="id">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="host" :label="$t('domainDetail.host')" width="180">
              <template #default="{ row }">
                <span>{{ row.host }}</span>
                <el-tag v-if="row.own_node_id" type="success" size="small" style="margin-left:4px">{{ $t('domainDetail.materialized') }}</el-tag>
                <template v-else-if="row.host !== '@'">
                  <el-tag type="info" size="small" style="margin-left:4px">{{ $t('domainDetail.implicit') }}</el-tag>
                  <el-button link type="primary" size="small" @click="handleConvertToNode(row.host)" style="margin-left:4px">{{ $t('domainDetail.convertToNode') }}</el-button>
                </template>
              </template>
            </el-table-column>
            <el-table-column prop="record_type" :label="$t('domainDetail.type')" width="80" />
            <el-table-column prop="value" :label="$t('domainDetail.value')" show-overflow-tooltip />
            <el-table-column prop="priority" :label="$t('domainDetail.priority')" width="70">
              <template #default="{ row }">{{ row.priority ?? '-' }}</template>
            </el-table-column>
            <el-table-column prop="ttl" :label="$t('domainDetail.ttl')" width="70" />
            <el-table-column prop="line" :label="$t('domainDetail.line')" width="80">
              <template #default="{ row }">{{ row.line || 'default' }}</template>
            </el-table-column>
            <el-table-column :label="$t('common.enabled')" width="70">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="(val) => handleToggle(row.id, val)" />
              </template>
            </el-table-column>
            <el-table-column prop="sync_status" :label="$t('domainDetail.syncStatus')" width="90">
              <template #default="{ row }">
                <el-tag :type="statusType(row.sync_status)" size="small">{{ statusLabel(row.sync_status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="last_resolved_at" :label="$t('domainDetail.lastResolved')" width="160" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.last_resolved_at || '—' }}
              </template>
            </el-table-column>
            <el-table-column :label="$t('common.actions')" min-width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="editRecord(row)">{{ $t('common.edit') }}</el-button>
                <el-button v-if="row.sync_status === 'failed' && auth.isAdmin" link type="warning" size="small" @click="handleRetrySync(row.id)">{{ $t('common.retry') }}</el-button>
                <el-button v-if="row.host !== '@'" link type="warning" size="small" @click="openTransferRecord(row)">{{ $t('domainDetail.transfer') }}</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteRecord(row.id)">{{ $t('common.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && records.length === 0" :description="$t('domainDetail.noRecords')" />

          <!-- pagination -->
          <div class="pagination-bar" v-if="total > 0">
            <el-pagination
              v-model:current-page="pagination.page"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              @current-change="loadRecords"
              @size-change="loadRecords"
            />
          </div>
        </el-card>
      </el-tab-pane>

      <!-- Authorization Tab -->
      <el-tab-pane :label="$t('domainDetail.tabAuthorization')" name="authorization">
        <!-- Permission list -->
        <el-card>
          <template #header>
            <div class="card-header">
              <span>{{ $t('domainDetail.permManagement') }}</span>
              <el-button size="small" type="primary" @click="showGrantPerm = true">{{ $t('domainDetail.grant') }}</el-button>
            </div>
          </template>
          <div v-if="permissions.length === 0" style="color:#909399;font-size:13px">{{ $t('domainDetail.noDelegatedPerms') }}</div>
          <div v-for="p in permissions" :key="p.id" class="perm-item">
            <div class="perm-info">
              <el-avatar v-if="p.user?.avatar" :src="p.user.avatar" :size="24" />
              <el-avatar v-else :size="24">{{ (p.user?.username || '?')[0]?.toUpperCase() }}</el-avatar>
              <span class="perm-user">{{ p.user?.username || 'User #' + p.user_id }}</span>
              <el-tag :type="permTagType(p.permission_level)" size="small">{{ permLabel(p.permission_level) }}</el-tag>
              <el-tag v-if="p.status === 'pending_return'" type="warning" size="small" style="margin-left:4px">{{ $t('domainDetail.pendingReturn') }}</el-tag>
              <el-tag v-if="p.status === 'returned'" type="info" size="small" style="margin-left:4px">{{ $t('domainDetail.returned') }}</el-tag>
            </div>
            <div class="perm-detail" v-if="p.allowed_types || p.allowed_ips || p.host_prefix || p.host_rules || p.max_depth">
              <span v-if="hostRulesSummary(p)" class="perm-restrict">{{ hostRulesSummary(p) }}</span>
              <span v-else-if="p.host_prefix" class="perm-restrict">{{ $t('domainDetail.prefixLabel') }} {{ p.host_prefix }}</span>
              <span v-if="p.max_depth" class="perm-restrict">{{ $t('domainDetail.depthLabel') }} {{ p.max_depth }}</span>
              <span v-if="p.allowed_types" class="perm-restrict">{{ $t('domainDetail.typesLabel') }} {{ p.allowed_types }}</span>
              <span v-if="p.allowed_ips" class="perm-restrict">{{ $t('domainDetail.ipsLabel') }} {{ p.allowed_ips }}</span>
            </div>
            <div class="perm-actions">
              <el-button v-if="p.status === 'active'" link type="danger" size="small" @click="handleRevokePerm(p.user_id)">{{ $t('domainDetail.revoke') }}</el-button>
              <el-button v-if="p.status === 'active' && p.permission_level !== 'read'" link type="warning" size="small" @click="handleRevokeRequest(p.user_id)">{{ $t('domainDetail.revokeRequest') }}</el-button>
            </div>
          </div>
        </el-card>

        <!-- Pending records -->
        <el-card style="margin-top:16px" v-if="pendingRecords.length > 0">
          <template #header>
            <div class="card-header">
              <span>{{ $t('domainDetail.pendingRecords') }}</span>
              <div>
                <el-button size="small" type="primary" @click="handleAssignSelected">{{ $t('domainDetail.assignSelected') }}</el-button>
                <el-button size="small" type="danger" @click="handleDeleteSelectedPending">{{ $t('domainDetail.deleteSelected') }}</el-button>
              </div>
            </div>
          </template>
          <el-table :data="pendingRecords" stripe @selection-change="handlePendingSelectionChange" row-key="id">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="host" :label="$t('domainDetail.host')" width="120" />
            <el-table-column prop="record_type" :label="$t('domainDetail.type')" width="80" />
            <el-table-column prop="value" :label="$t('domainDetail.value')" show-overflow-tooltip />
            <el-table-column prop="pending_group" :label="$t('domainDetail.group')" width="160" show-overflow-tooltip />
            <el-table-column :label="$t('common.actions')" width="120">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="handleAssignSingle(row.id)">{{ $t('domainDetail.assignToSelf') }}</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteSinglePending(row.id)">{{ $t('common.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- grant permission dialog -->
    <el-dialog v-model="showGrantPerm" :title="$t('domainDetail.grantUser')" width="520px" destroy-on-close>
      <el-form :model="grantForm" label-width="100px">
        <el-form-item :label="$t('domainDetail.users')">
          <el-select v-model="grantForm.target_user_ids" multiple filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUsers')" style="width:100%" collapse-tags collapse-tags-tooltip>
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.permLevel')">
          <el-select v-model="grantForm.level" style="width:100%">
            <el-option :label="$t('domainDetail.readOnlyOption')" value="read" />
            <el-option :label="$t('domainDetail.readWriteOption')" value="write" />
            <el-option :label="$t('domainDetail.adminOption')" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.hostRestrictions')">
          <div class="host-rules">
            <div v-for="(rule, idx) in grantForm.host_rules" :key="idx" class="host-rule-row">
              <el-select v-model="rule.type" style="width:110px">
                <el-option :label="$t('domainDetail.ruleExact')" value="exact" />
                <el-option :label="$t('domainDetail.rulePrefix')" value="prefix" />
                <el-option :label="$t('domainDetail.ruleSuffix')" value="suffix" />
                <el-option :label="$t('domainDetail.ruleContains')" value="contains" />
                <el-option :label="$t('domainDetail.ruleRegex')" value="regex" />
              </el-select>
              <el-input v-model="rule.value" placeholder="e.g. www" style="width:200px" />
              <el-button @click="removeHostRule(idx)" :icon="'Delete'" circle size="small" />
            </div>
            <el-button size="small" @click="addHostRule">{{ $t('domainDetail.addRule') }}</el-button>
          </div>
          <div style="color:#909399;font-size:12px;margin-top:4px">
            {{ $t('domainDetail.hostRulesDesc') }}
          </div>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.maxDepth')">
          <el-input-number v-model="grantForm.max_depth" :min="1" :max="10" :placeholder="$t('domainDetail.maxDepthPlaceholder')" />
          <div style="color:#909399;font-size:12px;margin-top:4px">{{ $t('domainDetail.maxDepthDesc') }}</div>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.recordTypes')">
          <el-checkbox-group v-model="grantForm.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t.value" :label="t.value">{{ t.value }}</el-checkbox>
          </el-checkbox-group>
          <div style="color:#909399;font-size:12px;margin-top:4px">{{ $t('domainDetail.recordTypesDesc') }}</div>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.ipRestriction')">
          <el-input v-model="grantForm.allowed_ips_text" type="textarea" :rows="2" :placeholder="$t('domainDetail.ipPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGrantPerm = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleGrantPerm">{{ $t('domainDetail.confirmGrant') }}</el-button>
      </template>
    </el-dialog>

    <!-- add record -->
    <el-dialog v-model="showAddRecord" :title="$t('domainDetail.addDnsRecord')" width="520px" destroy-on-close>
      <el-form :model="recordForm" label-width="80px">
        <el-form-item :label="$t('domainDetail.host')">
          <el-input v-model="recordForm.host" :placeholder="$t('domainDetail.atRootHint')" />
        </el-form-item>
        <el-form-item :label="$t('domainDetail.recordType')">
          <el-select v-model="recordForm.record_type" style="width:100%" @change="onTypeChange">
            <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'MX'" :label="$t('domainDetail.priority')">
          <el-input-number v-model="recordForm.priority" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'SRV'" :label="$t('domainDetail.srvValue')">
          <div style="display:flex;gap:8px;width:100%">
            <el-input-number v-model="srvForm.priority" :min="0" :max="65535" :placeholder="$t('domainDetail.srvPriority')" style="flex:1" />
            <el-input-number v-model="srvForm.weight" :min="0" :max="65535" :placeholder="$t('domainDetail.srvWeight')" style="flex:1" />
            <el-input-number v-model="srvForm.port" :min="0" :max="65535" :placeholder="$t('domainDetail.srvPort')" style="flex:1" />
          </div>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'SRV'" :label="$t('domainDetail.srvTarget')">
          <el-input v-model="srvForm.target" :placeholder="$t('domainDetail.srvTargetPlaceholder')" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" :label="$t('domainDetail.caaFlag')">
          <el-input-number v-model="caaForm.flag" :min="0" :max="255" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" :label="$t('domainDetail.caaTag')">
          <el-select v-model="caaForm.tag" style="width:100%">
            <el-option :label="$t('domainDetail.caaIssue')" value="issue" />
            <el-option :label="$t('domainDetail.caaIssueWild')" value="issuewild" />
            <el-option :label="$t('domainDetail.caaIodef')" value="iodef" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" :label="$t('domainDetail.caaValue')">
          <el-input v-model="caaForm.value" :placeholder="$t('domainDetail.caaValuePlaceholder')" />
        </el-form-item>
        <el-form-item v-if="!['SRV', 'CAA'].includes(recordForm.record_type)" :label="$t('domainDetail.value')">
          <el-input v-model="recordForm.value" :placeholder="valuePlaceholder" />
        </el-form-item>
        <el-form-item :label="$t('domainDetail.ttl')">
          <el-select v-model="recordForm.ttl" style="width:100%">
            <el-option :label="$t('domainDetail.ttl60')" :value="60" />
            <el-option :label="$t('domainDetail.ttl300')" :value="300" />
            <el-option :label="$t('domainDetail.ttl600')" :value="600" />
            <el-option :label="$t('domainDetail.ttl1800')" :value="1800" />
            <el-option :label="$t('domainDetail.ttl3600')" :value="3600" />
            <el-option :label="$t('domainDetail.ttl43200')" :value="43200" />
            <el-option :label="$t('domainDetail.ttl86400')" :value="86400" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.line')">
          <el-select v-model="recordForm.line" style="width:100%">
            <el-option :label="$t('domainDetail.lineDefault')" value="default" />
            <el-option :label="$t('domainDetail.lineTelecom')" value="telecom" />
            <el-option :label="$t('domainDetail.lineUnicom')" value="unicom" />
            <el-option :label="$t('domainDetail.lineMobile')" value="mobile" />
            <el-option :label="$t('domainDetail.lineOverseas')" value="overseas" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddRecord = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleAddRecord">{{ $t('domainDetail.add') }}</el-button>
      </template>
    </el-dialog>

    <!-- edit record -->
    <el-dialog v-model="showEditRecord" :title="$t('domainDetail.editDnsRecord')" width="480px" destroy-on-close>
      <el-form :model="editForm" label-width="80px">
        <el-form-item :label="$t('domainDetail.host')">
          <el-input v-model="editForm.host" disabled />
        </el-form-item>
        <el-form-item :label="$t('domainDetail.recordType')">
          <el-input v-model="editForm.record_type" disabled />
        </el-form-item>
        <el-form-item v-if="editForm.record_type === 'MX'" :label="$t('domainDetail.priority')">
          <el-input-number v-model="editForm.priority" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item :label="$t('domainDetail.value')">
          <el-input v-model="editForm.value" />
        </el-form-item>
        <el-form-item :label="$t('domainDetail.ttl')">
          <el-select v-model="editForm.ttl" style="width:100%">
            <el-option :label="$t('domainDetail.ttl60')" :value="60" />
            <el-option :label="$t('domainDetail.ttl300')" :value="300" />
            <el-option :label="$t('domainDetail.ttl600')" :value="600" />
            <el-option :label="$t('domainDetail.ttl1800')" :value="1800" />
            <el-option :label="$t('domainDetail.ttl3600')" :value="3600" />
            <el-option :label="$t('domainDetail.ttl43200')" :value="43200" />
            <el-option :label="$t('domainDetail.ttl86400')" :value="86400" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditRecord = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdateRecord">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- import records -->
    <el-dialog v-model="showImport" :title="$t('domainDetail.importDnsRecord')" width="480px" destroy-on-close>
      <el-tabs v-model="importTab">
        <el-tab-pane :label="$t('domainDetail.importJson')" name="json">
          <el-input v-model="importJson" type="textarea" :rows="10" placeholder='[{"host":"@","record_type":"A","value":"1.2.3.4","ttl":600}]' />
        </el-tab-pane>
        <el-tab-pane :label="$t('domainDetail.importCsv')" name="csv">
          <el-upload
            :auto-upload="false"
            :limit="1"
            accept=".csv"
            :on-change="handleCsvFile"
            :file-list="csvFileList"
          >
            <el-button size="small">{{ $t('domainDetail.selectCsvFile') }}</el-button>
            <template #tip>
              <div class="el-upload__tip">{{ $t('domainDetail.csvFormat') }}</div>
            </template>
          </el-upload>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="showImport = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleImport">{{ $t('domainDetail.import') }}</el-button>
      </template>
    </el-dialog>

    <!-- transfer domain -->
    <el-dialog v-model="showTransfer" :title="$t('domainDetail.transferDomain')" width="400px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('domainDetail.transferWarning') }}
      </el-alert>
      <el-form :model="transferForm" label-width="80px">
        <el-form-item :label="$t('domainDetail.targetUser')">
          <el-select v-model="transferForm.target_user_id" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTransfer = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleTransfer">{{ $t('domainDetail.confirmTransfer') }}</el-button>
      </template>
    </el-dialog>

    <!-- batch transfer dialog -->
    <el-dialog v-model="showBatchTransfer" :title="$t('domainDetail.transferSubdomain')" width="560px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('domainDetail.transferConsequenceWarning') }}
      </el-alert>
      <div style="margin-bottom:16px">
        <div style="font-weight:600;margin-bottom:8px">{{ $t('domainDetail.transferHostsSummary') }}</div>
        <div v-for="group in batchTransferGroups" :key="group.host" class="transfer-host-item">
          <div style="display:flex;align-items:center;gap:8px">
            <span style="font-weight:500">{{ group.host }}</span>
            <el-tag size="small">{{ $t('domainDetail.hostRecordsCount', { count: group.count }) }}</el-tag>
            <el-tag v-if="group.blocked" type="danger" size="small">{{ $t('domainDetail.hostStatusBlocked') }}</el-tag>
            <el-tag v-else-if="group.materialized" type="success" size="small">{{ $t('domainDetail.hostStatusMaterialized') }}</el-tag>
            <el-tag v-else type="info" size="small">{{ $t('domainDetail.hostStatusImplicit') }}</el-tag>
          </div>
          <div v-if="group.blocked" style="font-size:12px;color:#909399;margin-top:2px">
            {{ $t('domainDetail.transferRootHint') }}
          </div>
        </div>
      </div>
      <el-form label-width="80px">
        <el-form-item :label="$t('domainDetail.targetUser')">
          <el-select v-model="batchTransferForm.target_user_id" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBatchTransfer = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleBatchTransfer">{{ $t('domainDetail.batchTransfer') }}</el-button>
      </template>
    </el-dialog>

    <!-- single transfer dialog -->
    <el-dialog v-model="showTransferRecord" :title="$t('domainDetail.transferSubdomainTitle', { host: transferRecordForm.host })" width="480px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('domainDetail.transferConsequenceWarning') }}
      </el-alert>
      <div style="margin-bottom:16px">
        <div style="font-weight:600;margin-bottom:4px">{{ transferRecordForm.host }}</div>
        <div style="font-size:13px;color:#606266">
          {{ $t('domainDetail.hostRecordsCount', { count: transferRecordForm.recordCount }) }}
        </div>
      </div>
      <el-form label-width="80px">
        <el-form-item :label="$t('domainDetail.targetUser')">
          <el-select v-model="transferRecordForm.target_user_id" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTransferRecord = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleSingleTransfer">{{ $t('domainDetail.transfer') }}</el-button>
      </template>
    </el-dialog>

    <!-- conflict detection dialog -->
    <el-dialog v-model="showConflictDialog" :title="$t('domainDetail.conflictTitle')" width="520px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('domainDetail.conflictDesc') }}
      </el-alert>
      <el-descriptions v-if="conflictRecord" :column="2" size="small" border style="margin-bottom:16px">
        <el-descriptions-item :label="$t('domainDetail.host')">{{ conflictRecord.host }}</el-descriptions-item>
        <el-descriptions-item :label="$t('domainDetail.type')">{{ conflictRecord.type }}</el-descriptions-item>
        <el-descriptions-item :label="$t('domainDetail.value')" :span="2">{{ conflictRecord.value }}</el-descriptions-item>
        <el-descriptions-item :label="$t('domainDetail.ttl')">{{ conflictRecord.ttl }}</el-descriptions-item>
        <el-descriptions-item :label="$t('domainDetail.line')">{{ conflictRecord.line || 'default' }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="showConflictDialog = false">{{ $t('domainDetail.conflictCancel') }}</el-button>
        <el-button type="warning" @click="handleConflictCreateNew">{{ $t('domainDetail.conflictCreateNew') }}</el-button>
        <el-button type="primary" @click="handleConflictImport">{{ $t('domainDetail.conflictImport') }}</el-button>
      </template>
    </el-dialog>

    <!-- revoke permission - record handling -->
    <el-dialog v-model="showReturnDialog" :title="$t('domainDetail.returnDialogTitle')" width="480px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('domainDetail.returnWarning') }}
      </el-alert>
      <el-form :model="returnForm" label-width="80px">
        <el-form-item :label="$t('domainDetail.returnAction')">
          <el-radio-group v-model="returnForm.action">
            <el-radio value="keep">{{ $t('domainDetail.returnKeep') }}</el-radio>
            <el-radio value="delete">{{ $t('domainDetail.returnDelete') }}</el-radio>
            <el-radio value="transfer">{{ $t('domainDetail.returnTransfer') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="returnForm.action === 'transfer'" :label="$t('domainDetail.targetUser')">
          <el-select v-model="returnForm.target_user_id" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showReturnDialog = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleAcceptReturn">{{ $t('domainDetail.confirmReturn') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomain, transferDomain, deleteDomain, convertToNode, demoteNode, transferRecordsByHost } from '../api/domain'
import { getRecords, createRecord, updateRecord, deleteRecord, toggleRecord, batchDeleteRecords, batchToggleRecords, exportRecords, importRecords, checkRecordConflict } from '../api/record'
import { retrySync } from '../api/admin'
import { getPermissions, grantPermission, batchGrantPermission, revokePermission, revokeRequest, acceptReturn, getPendingRecords, assignPendingRecords, deletePendingRecords } from '../api/permission'
import { useAuthStore } from '../stores/auth'
import { searchAllUsers } from '../api/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useError } from '../composables/useError'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const { showError } = useError()
const domainId = route.params.id

const recordTypes = [
  { label: t('domainDetail.recordTypeA'), value: 'A' },
  { label: t('domainDetail.recordTypeAAAA'), value: 'AAAA' },
  { label: t('domainDetail.recordTypeCNAME'), value: 'CNAME' },
  { label: t('domainDetail.recordTypeALIAS'), value: 'ALIAS' },
  { label: t('domainDetail.recordTypeMX'), value: 'MX' },
  { label: t('domainDetail.recordTypeTXT'), value: 'TXT' },
  { label: t('domainDetail.recordTypeCAA'), value: 'CAA' },
  { label: t('domainDetail.recordTypeNS'), value: 'NS' },
  { label: t('domainDetail.recordTypeSRV'), value: 'SRV' },
]

const activeTab = ref('records')
const domain = ref(null)
const records = ref([])
const total = ref(0)
const loading = ref(false)
const selectedIds = ref([])
const selectedRows = ref([])

const showAddRecord = ref(false)
const showEditRecord = ref(false)
const showTransfer = ref(false)
const showImport = ref(false)
const importTab = ref('json')
const importJson = ref('')
const csvFile = ref(null)
const csvFileList = ref([])

const permissions = ref([])
const showGrantPerm = ref(false)
const grantForm = ref({ target_user_ids: [], level: 'read', allowed_types: [], allowed_ips_text: '', host_rules: [], host_prefix: '', max_depth: null })
const selectableUsers = ref([])
const searchingUsers = ref(false)

const filters = reactive({ host: '', record_type: '', value: '', enabled: undefined, sync_status: '' })
const pagination = reactive({ page: 1, pageSize: 20 })

const recordForm = ref({ host: '@', record_type: 'A', value: '', ttl: 600, priority: null, line: 'default' })
const editForm = ref({ id: null, host: '', record_type: '', value: '', ttl: 600, priority: null })
const srvForm = ref({ priority: 0, weight: 0, port: 0, target: '' })
const caaForm = ref({ flag: 0, tag: 'issue', value: '' })
const transferForm = ref({ target_user_id: null })

const pendingRecords = ref([])
const selectedPendingIds = ref([])
const showReturnDialog = ref(false)
const returnForm = ref({ action: 'keep', target_user_id: null })
const returnTargetUserId = ref(null)

const showBatchTransfer = ref(false)
const batchTransferForm = ref({ target_user_id: null })
const showTransferRecord = ref(false)
const transferRecordForm = ref({ host: '', recordCount: 0, target_user_id: null })

// Conflict detection state
const showConflictDialog = ref(false)
const conflictRecord = ref(null)
const pendingCreateData = ref(null)

const statusType = (s) => s === 'synced' ? 'success' : s === 'failed' ? 'danger' : s === 'disabled' ? 'info' : 'warning'
const statusLabel = (s) => ({ synced: t('common.synced'), failed: t('domainDetail.syncFailed'), pending: t('domainDetail.pendingSync'), disabled: t('domainDetail.disabled') }[s] || s)

const valuePlaceholder = computed(() => {
  const m = { A: t('domainDetail.placeholderA'), AAAA: t('domainDetail.placeholderAAAA'), CNAME: t('domainDetail.placeholderCNAME'), ALIAS: t('domainDetail.placeholderALIAS'), MX: t('domainDetail.placeholderMX'), TXT: t('domainDetail.placeholderTXT'), NS: t('domainDetail.placeholderNS') }
  return m[recordForm.value.record_type] || t('domainDetail.placeholderValue')
})

const loadRecords = async () => {
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (filters.host) params.host = filters.host
    if (filters.record_type) params.record_type = filters.record_type
    if (filters.value) params.value = filters.value
    if (filters.enabled !== undefined && filters.enabled !== '') params.enabled = filters.enabled
    if (filters.sync_status) params.sync_status = filters.sync_status
    const res = await getRecords(domainId, params)
    records.value = res.data.items
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

const loadDomain = async () => {
  const res = await getDomain(domainId)
  domain.value = res.data
}

const loadData = async () => {
  await Promise.all([loadDomain(), loadRecords(), loadPermissions(), loadPendingRecords()])
}

const onTypeChange = () => {
  recordForm.value.value = ''
  recordForm.value.priority = null
  srvForm.value = { priority: 0, weight: 0, port: 0, target: '' }
  caaForm.value = { flag: 0, tag: 'issue', value: '' }
}

const buildRecordValue = () => {
  const type = recordForm.value.record_type
  if (type === 'SRV') {
    return `${srvForm.value.priority} ${srvForm.value.weight} ${srvForm.value.port} ${srvForm.value.target}`
  }
  if (type === 'CAA') {
    return `${caaForm.value.flag} ${caaForm.value.tag} ${caaForm.value.value}`
  }
  return recordForm.value.value
}

const openAddRecord = () => {
  recordForm.value = { host: '@', record_type: 'A', value: '', ttl: 600, priority: null, line: 'default' }
  srvForm.value = { priority: 0, weight: 0, port: 0, target: '' }
  caaForm.value = { flag: 0, tag: 'issue', value: '' }
  showAddRecord.value = true
}

const handleAddRecord = async () => {
  const data = {
    host: recordForm.value.host,
    record_type: recordForm.value.record_type,
    value: buildRecordValue(),
    ttl: recordForm.value.ttl,
    line: recordForm.value.line,
  }
  if (recordForm.value.record_type === 'MX' && recordForm.value.priority != null) {
    data.priority = recordForm.value.priority
  }

  // Check for provider conflict first
  try {
    const res = await checkRecordConflict(domainId, { host: data.host, record_type: data.record_type })
    if (res.data?.has_conflict) {
      conflictRecord.value = res.data.existing_record
      pendingCreateData.value = data
      showAddRecord.value = false
      showConflictDialog.value = true
      return
    }
  } catch {
    // Conflict check failed -- proceed with creation anyway
  }

  await createRecord(domainId, data)
  ElMessage.success(t('domainDetail.addRecordSuccess'))
  showAddRecord.value = false
  loadRecords()
}

const handleConflictImport = async () => {
  const data = pendingCreateData.value
  data.value = conflictRecord.value.value
  data.provider_record_id = conflictRecord.value.record_id
  await createRecord(domainId, data)
  ElMessage.success(t('domainDetail.conflictImported'))
  showConflictDialog.value = false
  loadRecords()
}

const handleConflictCreateNew = async () => {
  await createRecord(domainId, pendingCreateData.value)
  ElMessage.success(t('domainDetail.addRecordSuccess'))
  showConflictDialog.value = false
  loadRecords()
}

const editRecord = (row) => {
  editForm.value = { id: row.id, host: row.host, record_type: row.record_type, value: row.value, ttl: row.ttl, priority: row.priority }
  showEditRecord.value = true
}

const handleUpdateRecord = async () => {
  const data = { value: editForm.value.value, ttl: editForm.value.ttl }
  if (editForm.value.record_type === 'MX' && editForm.value.priority != null) {
    data.priority = editForm.value.priority
  }
  await updateRecord(editForm.value.id, data)
  ElMessage.success(t('domainDetail.updateRecordSuccess'))
  showEditRecord.value = false
  loadRecords()
}

const handleToggle = async (id, enabled) => {
  await toggleRecord(id, { enabled })
  ElMessage.success(enabled ? t('domainDetail.recordEnabled') : t('domainDetail.recordDisabled'))
  loadRecords()
}

const handleDeleteRecord = async (id) => {
  await ElMessageBox.confirm(t('domainDetail.confirmDeleteRecord'), t('common.hint'), { type: 'warning' })
  await deleteRecord(id)
  ElMessage.success(t('domainDetail.recordDeleted'))
  loadRecords()
}

const handleSelectionChange = (rows) => {
  selectedIds.value = rows.map(r => r.id)
  selectedRows.value = rows
}

const handleBatchDelete = async () => {
  await ElMessageBox.confirm(t('domainDetail.confirmBatchDelete', { count: selectedIds.value.length }), t('domainDetail.batchDeleteTitle'), { type: 'warning' })
  await batchDeleteRecords(selectedIds.value)
  ElMessage.success(t('domainDetail.batchDeleteSuccess'))
  loadRecords()
}

const handleBatchToggle = async (enabled) => {
  await batchToggleRecords(selectedIds.value, enabled)
  ElMessage.success(enabled ? t('domainDetail.batchEnableSuccess') : t('domainDetail.batchDisableSuccess'))
  loadRecords()
}

const handleExport = async (format) => {
  const res = await exportRecords(domainId, format)
  if (format === 'csv') {
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = 'records.csv'
    a.click()
    URL.revokeObjectURL(url)
  } else {
    const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'records.json'
    a.click()
    URL.revokeObjectURL(url)
  }
}

const handleCsvFile = (file) => {
  csvFile.value = file.raw
}

const handleImport = async () => {
  if (importTab.value === 'json') {
    let data
    try {
      data = JSON.parse(importJson.value)
    } catch {
      ElMessage.error(t('domainDetail.jsonFormatError'))
      return
    }
    const res = await importRecords(domainId, data, 'json')
    const r = res.data
    ElMessage.success(t('domainDetail.importComplete', { created: r.created, skipped: r.skipped }))
    if (r.errors?.length) console.warn('Import errors:', r.errors)
  } else {
    if (!csvFile.value) {
      ElMessage.error(t('domainDetail.pleaseSelectCsv'))
      return
    }
    const res = await importRecords(domainId, csvFile.value, 'csv')
    const r = res.data
    ElMessage.success(t('domainDetail.importComplete', { created: r.created, skipped: r.skipped }))
    if (r.errors?.length) console.warn('Import errors:', r.errors)
  }
  showImport.value = false
  importJson.value = ''
  csvFile.value = null
  csvFileList.value = []
  loadRecords()
}

const handleRetrySync = async (id) => {
  await retrySync(id)
  ElMessage.success(t('domainDetail.resyncSuccess'))
  loadRecords()
}

const handleTransfer = async () => {
  await ElMessageBox.confirm(t('domainDetail.confirmTransferMsg'), t('domainDetail.confirmTransfer'), { type: 'warning' })
  await transferDomain(domainId, transferForm.value)
  ElMessage.success(t('domainDetail.transferSuccess'))
  showTransfer.value = false
  router.push('/dashboard')
}

const handleDeleteDomain = async () => {
  await ElMessageBox.confirm(t('domainDetail.confirmDeleteDomain'), t('common.confirmDelete'), { type: 'error' })
  await deleteDomain(domainId)
  ElMessage.success(t('domainDetail.deleteDomainSuccess'))
  router.push('/dashboard')
}

const loadPermissions = async () => {
  try {
    const res = await getPermissions(domainId)
    permissions.value = res.data || []
  } catch { /* no permission to view */ }
}

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: t('common.read'), write: t('common.write'), admin: t('common.admin') }[level] || level)

const handleGrantPerm = async () => {
  const data = {
    target_user_ids: grantForm.value.target_user_ids,
    level: grantForm.value.level,
  }
  if (grantForm.value.allowed_types.length > 0) {
    data.allowed_types = grantForm.value.allowed_types
  }
  if (grantForm.value.allowed_ips_text.trim()) {
    data.allowed_ips = grantForm.value.allowed_ips_text.trim().split('\n').map(s => s.trim()).filter(Boolean)
  }
  const filteredRules = grantForm.value.host_rules.filter(r => r.value.trim())
  if (filteredRules.length > 0) {
    data.host_rules = filteredRules
    const firstPrefix = filteredRules.find(r => r.type === 'prefix')
    if (firstPrefix) data.host_prefix = firstPrefix.value
  }
  if (grantForm.value.max_depth != null && grantForm.value.max_depth > 0) {
    data.max_depth = grantForm.value.max_depth
  }
  await batchGrantPermission(domainId, data)
  ElMessage.success(t('domainDetail.grantSuccess'))
  showGrantPerm.value = false
  grantForm.value = { target_user_ids: [], level: 'read', allowed_types: [], allowed_ips_text: '', host_rules: [], host_prefix: '', max_depth: null }
  loadPermissions()
}

const addHostRule = () => {
  grantForm.value.host_rules.push({ type: 'prefix', value: '' })
}
const removeHostRule = (idx) => {
  grantForm.value.host_rules.splice(idx, 1)
}
const hostRulesSummary = (p) => {
  let rules = null
  if (p.host_rules) {
    try { rules = typeof p.host_rules === 'string' ? JSON.parse(p.host_rules) : p.host_rules } catch { rules = null }
  }
  if (!rules || !Array.isArray(rules) || rules.length === 0) return ''
  const valid = rules.filter(r => r.value)
  if (valid.length === 0) return ''
  if (valid.length === 1) {
    const typeMap = { exact: t('domainDetail.ruleExact'), prefix: t('domainDetail.rulePrefix'), suffix: t('domainDetail.ruleSuffix'), contains: t('domainDetail.ruleContains'), regex: t('domainDetail.ruleRegex') }
    return (typeMap[valid[0].type] || valid[0].type) + ': ' + valid[0].value
  }
  return t('domainDetail.hostRestrictions') + ': ' + valid.length + ' ' + t('domainDetail.addRule').toLowerCase()
}

const handleRevokePerm = async (userId) => {
  await ElMessageBox.confirm(t('domainDetail.confirmRevoke'), t('domainDetail.revokePermTitle'), { type: 'warning' })
  await revokePermission(domainId, userId)
  ElMessage.success(t('domainDetail.revokeSuccess'))
  loadPermissions()
}

const handleRevokeRequest = async (userId) => {
  await ElMessageBox.confirm(t('domainDetail.confirmRevokeRequest'), t('domainDetail.revokeRequestTitle'), { type: 'warning' })
  try {
    await revokeRequest(domainId, userId, { skipErrorToast: true })
    ElMessage.success(t('domainDetail.revokeRequestSent'))
    loadPermissions()
  } catch (e) {
    showError(e.response?.data?.message || t('domainDetail.requestFailed'))
  }
}

const loadPendingRecords = async () => {
  try {
    const res = await getPendingRecords(domainId)
    pendingRecords.value = res.data || []
  } catch { /* no permission to view */ }
}

const handlePendingSelectionChange = (rows) => {
  selectedPendingIds.value = rows.map(r => r.id)
}

const handleAssignSingle = async (id) => {
  await assignPendingRecords(domainId, { record_ids: [id] })
  ElMessage.success(t('domainDetail.recordAssigned'))
  loadPendingRecords()
  loadRecords()
}

const handleDeleteSinglePending = async (id) => {
  await ElMessageBox.confirm(t('domainDetail.confirmDeletePending'), t('common.hint'), { type: 'warning' })
  await deletePendingRecords(domainId, { record_ids: [id] })
  ElMessage.success(t('domainDetail.recordDeleted'))
  loadPendingRecords()
}

const handleAssignSelected = async () => {
  if (selectedPendingIds.value.length === 0) {
    ElMessage.warning(t('domainDetail.pleaseSelectRecords'))
    return
  }
  await assignPendingRecords(domainId, { record_ids: selectedPendingIds.value })
  ElMessage.success(t('domainDetail.recordAssigned'))
  loadPendingRecords()
  loadRecords()
}

const handleDeleteSelectedPending = async () => {
  if (selectedPendingIds.value.length === 0) {
    ElMessage.warning(t('domainDetail.pleaseSelectRecords'))
    return
  }
  await ElMessageBox.confirm(t('domainDetail.confirmBatchDelete', { count: selectedPendingIds.value.length }), t('domainDetail.batchDeleteTitle'), { type: 'warning' })
  await deletePendingRecords(domainId, { record_ids: selectedPendingIds.value })
  ElMessage.success(t('domainDetail.recordDeleted'))
  loadPendingRecords()
}

const handleAcceptReturn = async () => {
  const data = { action: returnForm.value.action }
  if (returnForm.value.action === 'transfer' && returnForm.value.target_user_id) {
    data.target_user_id = returnForm.value.target_user_id
  }
  try {
    await acceptReturn(domainId, returnTargetUserId.value, data, { skipErrorToast: true })
    ElMessage.success(t('domainDetail.returnComplete'))
    showReturnDialog.value = false
    loadPermissions()
    loadPendingRecords()
    loadRecords()
  } catch (e) {
    showError(e.response?.data?.message || t('domainDetail.returnFailed'))
  }
}

const handleConvertToNode = async (host) => {
  try {
    await ElMessageBox.confirm(
      t('domainDetail.convertToNodeConfirm', { host }),
      t('domainDetail.convertToNode'),
      { type: 'warning' }
    )
    const res = await convertToNode(domainId, { host })
    const count = res.data?.affected_records ?? 0
    ElMessage.success(t('domainDetail.convertSuccess', { count }))
    loadRecords()
  } catch (e) {
    if (e !== 'cancel') {
      showError(e.response?.data?.message || t('common.error'))
    }
  }
}

const handleDemoteNode = async () => {
  try {
    await ElMessageBox.confirm(
      t('domainDetail.demoteConfirm'),
      t('domainDetail.demoteNode'),
      { type: 'warning' }
    )
    await demoteNode(domainId)
    ElMessage.success(t('domainDetail.demoteSuccess'))
    // Navigate to parent
    if (domain.value?.parent_id) {
      router.push(`/domains/${domain.value.parent_id}`)
    } else {
      router.push('/dashboard')
    }
  } catch (e) {
    if (e !== 'cancel') {
      showError(e.response?.data?.message || t('common.error'))
    }
  }
}

const searchUsersRemote = async (query) => {
  if (!query) { selectableUsers.value = []; return }
  searchingUsers.value = true
  try {
    const res = await searchAllUsers(query)
    selectableUsers.value = (res.data || []).filter(u => u.id !== auth.user?.id)
  } catch { /* ignore */ }
  searchingUsers.value = false
}

const groupSelectedByHost = () => {
  const groups = {}
  for (const row of selectedRows.value) {
    const host = row.host
    if (!groups[host]) {
      groups[host] = { host, count: 0, materialized: !!row.own_node_id, blocked: host === '@' }
    }
    groups[host].count++
  }
  return Object.values(groups)
}

const batchTransferGroups = computed(() => groupSelectedByHost())

const openTransferRecord = (row) => {
  const hostRecords = records.value.filter(r => r.host === row.host)
  transferRecordForm.value = { host: row.host, recordCount: hostRecords.length, target_user_id: null }
  showTransferRecord.value = true
}

const showTransferResults = (results, targetUserLabel) => {
  const successItems = results.filter(r => r.status === 'transferred')
  const errorItems = results.filter(r => r.status === 'error')

  let html = ''
  for (const r of successItems) {
    html += `<div style="color:#67c23a;margin-bottom:4px">&#10003; ${r.full_domain || r.host} &rarr; ${targetUserLabel}</div>`
  }
  for (const r of errorItems) {
    const reason = r.message || r.error || t('common.error')
    html += `<div style="color:#f56c6c;margin-bottom:4px">&#10007; ${r.host}: ${reason}</div>`
  }

  const title = errorItems.length > 0
    ? t('domainDetail.transferPartial')
    : t('domainDetail.transferCompleted')

  ElMessageBox.alert(html, title, { dangerouslyUseHTMLString: true, confirmButtonText: t('common.confirm') })
}

const handleBatchTransfer = async () => {
  const groups = batchTransferGroups.value.filter(g => !g.blocked)
  if (groups.length === 0) {
    ElMessage.warning(t('domainDetail.transferRootHint'))
    return
  }
  if (!batchTransferForm.value.target_user_id) {
    ElMessage.warning(t('domainDetail.searchUser'))
    return
  }

  await ElMessageBox.confirm(
    t('domainDetail.confirmBatchTransferMsg', { count: groups.length }),
    t('domainDetail.confirmBatchTransfer'),
    { type: 'warning' }
  )

  try {
    const res = await transferRecordsByHost(domainId, {
      hosts: groups.map(g => g.host),
      target_user_id: batchTransferForm.value.target_user_id,
    })
    const targetLabel = selectableUsers.value.find(u => u.id === batchTransferForm.value.target_user_id)
      ? `@${selectableUsers.value.find(u => u.id === batchTransferForm.value.target_user_id).username}`
      : `#${batchTransferForm.value.target_user_id}`
    showTransferResults(res.data.results || [], targetLabel)
    showBatchTransfer.value = false
    batchTransferForm.value = { target_user_id: null }
    loadRecords()
  } catch (e) {
    showError(e.response?.data?.message || t('common.error'))
  }
}

const handleSingleTransfer = async () => {
  if (!transferRecordForm.value.target_user_id) {
    ElMessage.warning(t('domainDetail.searchUser'))
    return
  }

  await ElMessageBox.confirm(
    t('domainDetail.confirmSingleTransferMsg', { host: transferRecordForm.value.host, count: transferRecordForm.value.recordCount }),
    t('domainDetail.confirmSingleTransfer'),
    { type: 'warning' }
  )

  try {
    const res = await transferRecordsByHost(domainId, {
      hosts: [transferRecordForm.value.host],
      target_user_id: transferRecordForm.value.target_user_id,
    })
    const targetLabel = selectableUsers.value.find(u => u.id === transferRecordForm.value.target_user_id)
      ? `@${selectableUsers.value.find(u => u.id === transferRecordForm.value.target_user_id).username}`
      : `#${transferRecordForm.value.target_user_id}`
    showTransferResults(res.data.results || [], targetLabel)
    showTransferRecord.value = false
    transferRecordForm.value = { host: '', recordCount: 0, target_user_id: null }
    loadRecords()
  } catch (e) {
    showError(e.response?.data?.message || t('common.error'))
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.domain-info-bar {
  margin-bottom: 16px;
}
.domain-info-bar :deep(.el-card__body) {
  padding: 12px 20px;
}
.info-bar-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.info-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.info-bar-domain {
  font-size: 18px;
  font-weight: 600;
}
.info-bar-meta {
  color: #909399;
  font-size: 13px;
}
.info-bar-right {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.domain-detail-tabs {
  margin-top: 0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f0f9ff;
  border-radius: 4px;
}
.pagination-bar {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.perm-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  flex-wrap: wrap;
  gap: 4px;
}
.perm-item:last-child {
  border-bottom: none;
}
.perm-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.perm-user {
  font-weight: 500;
  font-size: 13px;
}
.perm-detail {
  width: 100%;
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: #909399;
}
.perm-restrict {
  background: #f5f5f5;
  padding: 1px 6px;
  border-radius: 3px;
}
.host-rules {
  width: 100%;
}
.host-rule-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.transfer-host-item {
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 6px;
}
</style>
