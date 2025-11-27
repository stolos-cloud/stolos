import api from './api';

export async function getClusterInfo() {
    try {
        const response = await api.get('/cluster/info');
        return response.data;
    } catch (error) {
        console.error('Error fetching cluster info:', error);
        throw error;
    }
}