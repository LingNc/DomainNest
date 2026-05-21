import request from './request'

export const listFilterPresets = () => request.get('/filter-presets')
export const saveFilterPreset = (data) => request.post('/filter-presets', data)
export const deleteFilterPreset = (id) => request.delete(`/filter-presets/${id}`)
