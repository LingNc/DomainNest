import request from './request'

export const getDomains = () => request.get('/domains')
export const createDomain = (data) => request.post('/domains', data)
export const getDomain = (id) => request.get(`/domains/${id}`)
export const transferDomain = (id, data) => request.post(`/domains/${id}/transfer`, data)
export const deleteDomain = (id) => request.delete(`/domains/${id}`)

export const convertToNode = (parentId, data) => request.post(`/domains/${parentId}/nodes/convert`, data)
export const transferRecordsByHost = (parentId, data) => request.post(`/domains/${parentId}/records/transfer`, data)
export const demoteNode = (nodeId) => request.post(`/domains/${nodeId}/nodes/demote`)
export const getConversionLogs = (nodeId) => request.get(`/domains/${nodeId}/nodes/conversion-logs`)
