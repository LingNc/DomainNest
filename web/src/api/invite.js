import request from './request'

export const generateMyInviteCodes = (data) => request.post('/auth/invite-codes', data)
export const listMyInviteCodes = (params) => request.get('/auth/invite-codes', { params })
export const deleteMyInviteCode = (id) => request.delete(`/auth/invite-codes/${id}`)
export const batchDeleteMyInviteCodes = (data) => request.post('/auth/invite-codes/batch-delete', data)
