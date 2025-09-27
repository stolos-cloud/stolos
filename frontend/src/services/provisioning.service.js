import axios from 'axios';

export async function downloadISO(payload) {
    try {
        const response = await axios.post('/api/isos/generate', payload, {
            responseType: 'blob',
        });
        return {
            data: response.data,
            headers: response.headers
        }
    } catch (error) {
        console.error('Error downloading ISO:', error);
        throw error;
    }
}
