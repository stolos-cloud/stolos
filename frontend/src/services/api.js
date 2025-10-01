import axios from 'axios'
import router from '@/router'

/*---------------------------------------------------
    Axios Instance Configuration for authentication
----------------------------------------------------*/
const api = axios.create({timeout: 10000});

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

api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      router.push({ name: 'login', query : { message: 'sessionExpired' } });
    }
    return Promise.reject(error)
  }
)

export default api