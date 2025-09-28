import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Add request interceptor to include auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Add response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      // Could redirect to login here if needed
    }
    return Promise.reject(error)
  }
)

export class AuthService {
  async login(email, password) {
    try {
      const response = await api.post('/auth/login', {
        email,
        password
      })

      const { token, user } = response.data

      // Store token and user data
      localStorage.setItem('token', token)
      localStorage.setItem('user', JSON.stringify(user))

      return { token, user }
    } catch (error) {
      throw new Error(error.response?.data?.error || 'Login failed')
    }
  }

  async logout() {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  async getProfile() {
    try {
      const response = await api.get('/auth/profile')
      return response.data.user
    } catch (error) {
      throw new Error(error.response?.data?.error || 'Failed to get profile')
    }
  }

  async refreshToken() {
    try {
      const response = await api.post('/auth/refresh')
      const { token, user } = response.data

      localStorage.setItem('token', token)
      localStorage.setItem('user', JSON.stringify(user))

      return { token, user }
    } catch (error) {
      throw new Error(error.response?.data?.error || 'Token refresh failed')
    }
  }

  isAuthenticated() {
    return !!localStorage.getItem('token')
  }

  getUser() {
    const userData = localStorage.getItem('user')
    return userData ? JSON.parse(userData) : null
  }

  getToken() {
    return localStorage.getItem('token')
  }
}

export default new AuthService()
