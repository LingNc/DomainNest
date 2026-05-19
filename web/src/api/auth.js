import request from './request'

export const login = (data) => request.post('/auth/login', data)
export const register = (data) => request.post('/auth/register', data)
export const getProfile = () => request.get('/auth/profile')
export const updateProfile = (data) => request.put('/auth/profile', data)
export const resetToken = () => request.put('/auth/token')
export const changePassword = (data) => request.put('/auth/password', data)
export const forgotPassword = (data) => request.post('/auth/forgot-password', data)
export const resetPassword = (data) => request.post('/auth/reset-password', data)
export const checkUsername = (username) => request.get('/auth/check-username', { params: { username } })
export const uploadAvatar = (formData) => request.post('/auth/avatar', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
