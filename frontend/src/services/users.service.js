import api from './api'

export async function getUsers() {
    try {
        const response = await api.get('/api/users');
        console.log(response.data);
        
        return response.data;
    } catch (error) {
        console.error('Error fetching users:', error);
        throw error;
    }
}