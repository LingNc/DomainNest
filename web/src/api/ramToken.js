import request from './request'

export const listRAMTokens = () => request.get('/ram-tokens')
export const createRAMToken = (data) => request.post('/ram-tokens', data)
export const getRAMToken = (id) => request.get(`/ram-tokens/${id}`)
export const updateRAMToken = (id, data) => request.put(`/ram-tokens/${id}`, data)
export const resetRAMToken = (id) => request.post(`/ram-tokens/${id}/reset`)
export const deleteRAMToken = (id) => request.delete(`/ram-tokens/${id}`)
