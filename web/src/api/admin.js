import request from './request'

export const createRootDomain = (data) => request.post('/admin/domains', data)
export const listDomains = () => request.get('/admin/domains')
export const assignDomain = (id, data) => request.post(`/admin/domains/${id}/assign`, data)
export const listUsers = () => request.get('/admin/users')
export const listLogs = (params) => request.get('/admin/logs', { params })
export const retrySync = (id) => request.post(`/admin/records/${id}/sync`)
