import request from './request'

export const getPermissions = (nodeId) => request.get(`/domains/${nodeId}/permissions`)
export const grantPermission = (nodeId, data) => request.post(`/domains/${nodeId}/permissions`, data)
export const revokePermission = (nodeId, userId) => request.delete(`/domains/${nodeId}/permissions/${userId}`)
export const getMyPermissions = () => request.get('/auth/permissions')
