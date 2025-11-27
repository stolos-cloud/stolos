import api from './api';

export async function getScaffolds() {
    try {
        const response = await api.get('/scaffolds');
        return response.data;
    } catch (error) {
        console.error('Error fetching scaffolds:', error);
        throw error;
    }
}
