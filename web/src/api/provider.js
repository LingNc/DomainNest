import request from './request'

export const listProviders = () => request.get('/providers')
export const createProvider = (data, config) => request.post('/providers', data, config)
export const getProvider = (id) => request.get(`/providers/${id}`)
export const updateProvider = (id, data, config) => request.put(`/providers/${id}`, data, config)
export const deleteProvider = (id, config) => request.delete(`/providers/${id}`, config)
export const listProviderDomains = (id, config) => request.get(`/providers/${id}/domains`, config)
export const claimDomain = (id, data, config) => request.post(`/providers/${id}/claim`, data, config)
export const reclaimDomain = (providerId, domainId) => request.post(`/providers/${providerId}/domains/${domainId}/reclaim`)
