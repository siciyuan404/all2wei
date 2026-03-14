import api from './axios';

export const register = (username, password) =>
  api.post('/register', { username, password });

export const login = (username, password) =>
  api.post('/login', { username, password });

export const getProfile = () =>
  api.get('/profile');
