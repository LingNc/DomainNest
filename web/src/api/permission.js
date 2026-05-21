import request from './request'

export const getPermissions = (nodeId) => request.get(`/domains/${nodeId}/permissions`)
export const grantPermission = (nodeId, data) => request.post(`/domains/${nodeId}/permissions`, data)
export const revokePermission = (nodeId, userId) => request.delete(`/domains/${nodeId}/permissions/${userId}`)
export const batchGrantPermission = (nodeId, data) => request.post(`/domains/${nodeId}/permissions/batch`, data)
export const getMyPermissions = () => request.get('/auth/permissions')
export const revokeRequest = (nodeId, userId, config) => request.post(`/domains/${nodeId}/permissions/${userId}/revoke-request`, undefined, config)
export const acceptReturn = (nodeId, userId, data, config) => request.post(`/domains/${nodeId}/permissions/${userId}/accept-return`, data, config)
export const rejectReturn = (nodeId, userId) => request.post(`/domains/${nodeId}/permissions/${userId}/reject-return`)
export const getPendingRecords = (nodeId) => request.get(`/domains/${nodeId}/pending-records`)
export const assignPendingRecords = (nodeId, data) => request.post(`/domains/${nodeId}/pending-records/assign`, data)
export const deletePendingRecords = (nodeId, data) => request.post(`/domains/${nodeId}/pending-records/delete`, data)
