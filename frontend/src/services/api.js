import axios from 'axios';
import router from '@/router';
import { StorageService } from './storage.service';

/*---------------------------------------------------
    Axios Instance Configuration for authentication
----------------------------------------------------*/
const api = axios.create({ timeout: 10000 });
export const API_BASE_VERSION = '/api/v1';

api.interceptors.request.use(
    config => {
        const token = StorageService.get('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    error => {
        return Promise.reject(error);
    }
);

api.interceptors.response.use(
    response => {
        return response;
    },
    error => {
        if (error.response?.status === 401) {
            StorageService.remove('token');
            StorageService.remove('user');
            router.push({ name: 'login', query: { message: 'sessionExpired' } });
        }
        return Promise.reject(error);
    }
);

const httpBaseURL = import.meta.env.VITE_API_BASE_URL
    ? new URL(API_BASE_VERSION, import.meta.env.VITE_API_BASE_URL).toString()
    : API_BASE_VERSION;
api.defaults.baseURL = httpBaseURL;

const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
export const WS_BASE_URL = import.meta.env.VITE_API_BASE_URL
    ? httpBaseURL.replace(/^http/, 'ws')
    : `${wsProtocol}//${window.location.host}${API_BASE_VERSION}`;

export default api;
