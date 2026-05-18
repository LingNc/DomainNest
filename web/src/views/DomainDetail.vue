<!-- web/src/views/DomainDetail.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-row :gutter="20">
        <el-col :span="16">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>DNS Records - {{ domain?.full_domain }}</span>
                <el-button type="primary" size="small" @click="showAddRecord = true">Add Record</el-button>
              </div>
            </template>
            <el-table :data="records" stripe>
              <el-table-column prop="host" label="Host" width="120" />
              <el-table-column prop="record_type" label="Type" width="80" />
              <el-table-column prop="value" label="Value" />
              <el-table-column prop="ttl" label="TTL" width="80" />
              <el-table-column prop="sync_status" label="Status" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.sync_status === 'synced' ? 'success' : row.sync_status === 'failed' ? 'danger' : 'warning'" size="small">
                    {{ row.sync_status }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="Actions" width="150">
                <template #default="{ row }">
                  <el-button size="small" @click="editRecord(row)">Edit</el-button>
                  <el-button size="small" type="danger" @click="handleDeleteRecord(row.id)">Delete</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card>
            <template #header>Actions</template>
            <el-button type="primary" @click="showCreateChild = true" style="width:100%;margin-bottom:10px">Create Subdomain</el-button>
            <el-button type="warning" @click="showTransfer = true" style="width:100%;margin-bottom:10px">Transfer Domain</el-button>
            <el-button type="danger" @click="handleDeleteDomain" style="width:100%">Delete Domain</el-button>
          </el-card>
        </el-col>
      </el-row>

      <!-- Add Record Dialog -->
      <el-dialog v-model="showAddRecord" title="Add DNS Record">
        <el-form :model="recordForm">
          <el-form-item label="Host">
            <el-input v-model="recordForm.host" placeholder="@ for root" />
          </el-form-item>
          <el-form-item label="Type">
            <el-select v-model="recordForm.record_type">
              <el-option label="A" value="A" />
              <el-option label="AAAA" value="AAAA" />
              <el-option label="CNAME" value="CNAME" />
              <el-option label="TXT" value="TXT" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="Value">
            <el-input v-model="recordForm.value" />
          </el-form-item>
          <el-form-item label="TTL">
            <el-input-number v-model="recordForm.ttl" :min="60" :max="86400" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showAddRecord = false">Cancel</el-button>
          <el-button type="primary" @click="handleAddRecord">Add</el-button>
        </template>
      </el-dialog>

      <!-- Edit Record Dialog -->
      <el-dialog v-model="showEditRecord" title="Edit DNS Record">
        <el-form :model="editForm">
          <el-form-item label="Host">
            <el-input v-model="editForm.host" disabled />
          </el-form-item>
          <el-form-item label="Type">
            <el-input v-model="editForm.record_type" disabled />
          </el-form-item>
          <el-form-item label="Value">
            <el-input v-model="editForm.value" />
          </el-form-item>
          <el-form-item label="TTL">
            <el-input-number v-model="editForm.ttl" :min="60" :max="86400" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showEditRecord = false">Cancel</el-button>
          <el-button type="primary" @click="handleUpdateRecord">Save</el-button>
        </template>
      </el-dialog>

      <!-- Create Child Dialog -->
      <el-dialog v-model="showCreateChild" title="Create Subdomain">
        <el-form :model="childForm">
          <el-form-item label="Host">
            <el-input v-model="childForm.host" placeholder="subdomain name" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showCreateChild = false">Cancel</el-button>
          <el-button type="primary" @click="handleCreateChild">Create</el-button>
        </template>
      </el-dialog>

      <!-- Transfer Dialog -->
      <el-dialog v-model="showTransfer" title="Transfer Domain">
        <el-form :model="transferForm">
          <el-form-item label="Target User ID">
            <el-input-number v-model="transferForm.target_user_id" :min="1" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showTransfer = false">Cancel</el-button>
          <el-button type="warning" @click="handleTransfer">Transfer</el-button>
        </template>
      </el-dialog>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDomain, createDomain, transferDomain, deleteDomain } from '../api/domain'
import { getRecords, createRecord, updateRecord, deleteRecord } from '../api/record'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const domainId = route.params.id

const domain = ref(null)
const records = ref([])
const showAddRecord = ref(false)
const showEditRecord = ref(false)
const showCreateChild = ref(false)
const showTransfer = ref(false)

const recordForm = ref({ host: '@', record_type: 'A', value: '', ttl: 600 })
const editForm = ref({ id: null, host: '', record_type: '', value: '', ttl: 600 })
const childForm = ref({ host: '' })
const transferForm = ref({ target_user_id: 1 })

const loadData = async () => {
  const [domainRes, recordsRes] = await Promise.all([
    getDomain(domainId),
    getRecords(domainId)
  ])
  domain.value = domainRes.data
  records.value = recordsRes.data
}

const handleAddRecord = async () => {
  await createRecord(domainId, recordForm.value)
  showAddRecord.value = false
  recordForm.value = { host: '@', record_type: 'A', value: '', ttl: 600 }
  loadData()
}

const editRecord = (row) => {
  editForm.value = { id: row.id, host: row.host, record_type: row.record_type, value: row.value, ttl: row.ttl }
  showEditRecord.value = true
}

const handleUpdateRecord = async () => {
  await updateRecord(editForm.value.id, { value: editForm.value.value, ttl: editForm.value.ttl })
  showEditRecord.value = false
  loadData()
}

const handleDeleteRecord = async (id) => {
  await ElMessageBox.confirm('Delete this record?')
  await deleteRecord(id)
  loadData()
}

const handleCreateChild = async () => {
  await createDomain({ parent_id: parseInt(domainId), host: childForm.value.host })
  showCreateChild.value = false
  router.push('/dashboard')
}

const handleTransfer = async () => {
  await ElMessageBox.confirm('Transfer this domain and all subdomains?')
  await transferDomain(domainId, transferForm.value)
  showTransfer.value = false
  router.push('/dashboard')
}

const handleDeleteDomain = async () => {
  await ElMessageBox.confirm('Delete this domain? Only works if no children or records.')
  await deleteDomain(domainId)
  router.push('/dashboard')
}

onMounted(loadData)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
