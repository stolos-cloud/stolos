import api from './api'
import { StorageService } from './storage.service'

export async function login(email, password) {
  try {
    const response = await api.post('/api/auth/login', {
      email,
      password
    })
    const { token, user } = response.data

    StorageService.set('token', token);
    StorageService.set('user', JSON.stringify(user));

    return { token, user }
  } catch (error) {
    throw new Error(error.response?.data?.error || 'failedLogin')
  }
}

export async function logout() {
  StorageService.remove('token');
  StorageService.remove('user');
}

export async function getProfile() {
  try {
    const response = await api.get('/api/auth/profile')
    return response.data.user
  } catch (error) {
    throw new Error(error.response?.data?.error || 'failedFetchProfile')
  }
}

export async function refreshToken() {
  try {
    const response = await api.post('/api/auth/refresh')
    const { token } = response.data

    StorageService.set('token', token)
    return { token }
  } catch (error) {
    throw new Error(error.response?.data?.error || 'failedRefreshToken')
  }
}

export async function initAuthenticationExistingToken() {
  try {
    const token = StorageService.get('token');
    if(!token) {
      return null;
    }
    const user = await getProfile();
    return { token, user };
  } catch (error) {
    throw new Error(error.response?.data?.error || 'failedInitAuth');
  }
}