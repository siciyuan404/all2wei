import api from './axios';

export const uploadMaterial = (formData, config = {}) =>
  api.post('/materials', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    ...config,
  });

export const getMaterials = (folder) => {
  const params = {};
  if (folder) params.folder = folder;
  return api.get('/materials', { params });
};

export const getFolders = () =>
  api.get('/materials/folders');

export const getMaterial = (id) =>
  api.get(`/materials/${id}`);

export const deleteMaterial = (id) =>
  api.delete(`/materials/${id}`);

export const getSubtitle = (id) =>
  api.get(`/materials/${id}/subtitle`);

export const syncMaterials = () =>
  api.post('/materials/sync');

export const getVideoStreamUrl = (id) => {
  const baseURL = api.defaults.baseURL.replace('/api', '');
  const token = localStorage.getItem('token');
  const tokenParam = token ? `?token=${token}` : '';
  return `${baseURL}/api/materials/${id}/stream${tokenParam}`;
};
