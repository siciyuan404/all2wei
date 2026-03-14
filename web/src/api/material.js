import api from './axios';

export const uploadMaterial = (formData) =>
  api.post('/materials', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });

export const getMaterials = () =>
  api.get('/materials');

export const getMaterial = (id) =>
  api.get(`/materials/${id}`);

export const deleteMaterial = (id) =>
  api.delete(`/materials/${id}`);

export const getSubtitle = (id) =>
  api.get(`/materials/${id}/subtitle`);

export const syncMaterials = () =>
  api.post('/materials/sync');

// 获取视频流 URL（使用代理解决跨域问题）
export const getVideoStreamUrl = (id) => {
  const baseURL = api.defaults.baseURL.replace('/api', '');
  const token = localStorage.getItem('token');
  const tokenParam = token ? `?token=${token}` : '';
  return `${baseURL}/api/materials/${id}/stream${tokenParam}`;
};