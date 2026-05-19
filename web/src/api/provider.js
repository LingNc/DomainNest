import request from './request'

export const listProviders = () => request.get('/providers')
export const createProvider = (data) => request.post('/providers', data)
export const getProvider = (id) => request.get(`/providers/${id}`)
export const updateProvider = (id, data) => request.put(`/providers/${id}`, data)
export const deleteProvider = (id) => request.delete(`/providers/${id}`)
export const listProviderDomains = (id) => request.get(`/providers/${id}/domains`)
export const claimDomain = (id, data) => request.post(`/providers/${id}/claim`, data)
