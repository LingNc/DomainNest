import request from './request'

export const getRecords = (nodeId, params) => request.get(`/domains/${nodeId}/records`, { params })
export const createRecord = (nodeId, data) => request.post(`/domains/${nodeId}/records`, data)
export const updateRecord = (id, data) => request.put(`/records/${id}`, data)
export const deleteRecord = (id) => request.delete(`/records/${id}`)
export const toggleRecord = (id, data) => request.put(`/records/${id}/toggle`, data)
export const batchDeleteRecords = (ids) => request.post('/records/batch-delete', { ids })
export const batchToggleRecords = (ids, enabled) => request.post('/records/batch-toggle', { ids, enabled })
export const exportRecords = (nodeId, format) => request.get(`/domains/${nodeId}/records/export`, { params: { format }, responseType: format === 'csv' ? 'blob' : 'json' })
export const importRecords = (nodeId, data, format) => {
  if (format === 'csv') {
    const formData = new FormData()
    formData.append('file', data)
    return request.post(`/domains/${nodeId}/records/import?format=csv`, formData, { headers: { 'Content-Type': 'multipart/form-data' } })
  }
  return request.post(`/domains/${nodeId}/records/import`, data)
}

export const checkRecordConflict = (nodeId, data) =>
  request.post(`/domains/${nodeId}/records/check-conflict`, data)

export const batchTagRecords = (data) => request.put('/records/batch-tag', data)

export const syncRecord = (id) => request.post(`/records/${id}/sync`)
export const adoptRecord = (id) => request.put(`/records/${id}/adopt`)
