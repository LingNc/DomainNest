import request from './request'

export const listTrash = (params) => request.get('/trash', { params })
export const trashRecord = (id) => request.post(`/trash/${id}/trash`)
export const restoreRecord = (id) => request.post(`/trash/${id}/restore`)
export const permanentDelete = (id) => request.delete(`/trash/${id}`)
export const emptyTrash = () => request.post('/trash/empty')
export const batchTrash = (record_ids) => request.post('/trash/batch-trash', { record_ids })
export const batchRestore = (record_ids) => request.post('/trash/batch-restore', { record_ids })
