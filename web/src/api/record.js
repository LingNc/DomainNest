import request from './request'

export const getRecords = (nodeId) => request.get(`/domains/${nodeId}/records`)
export const createRecord = (nodeId, data) => request.post(`/domains/${nodeId}/records`, data)
export const updateRecord = (id, data) => request.put(`/records/${id}`, data)
export const deleteRecord = (id) => request.delete(`/records/${id}`)
