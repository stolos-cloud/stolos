import api from './api';

export async function getNamespaces() {
    try {
        const response = await api.get('/namespaces');
        return response.data;
    } catch (error) {
        console.error('Error fetching namespaces:', error);
        throw error;
    }
}

export async function createNewNamespace(namespaceData) {
    try {
        const response = await api.post('/namespaces', namespaceData);
        return response.data;
    } catch (error) {
        console.error('Error creating namespace:', error);
        throw error;
    }
}

export async function getNamespaceDetails(namespaceId) {
    try {
        const response = await api.get(`/namespaces/${namespaceId}`);
        return response.data;
    } catch (error) {
        console.error(`Error fetching namespace with ID ${namespaceId}:`, error);
        throw error;
    }
}

export async function addUserIdToNamespace(namespaceId, user) {
    try {
        const response = await api.post(`/namespaces/${namespaceId}/users`, user);
        return response.data;
    } catch (error) {
        console.error(`Error adding user to namespace with ID ${namespaceId}:`, error);
        throw error;
    }
}

export async function deleteNamespaceById(namespaceId) {
    try {
        const response = await api.delete(`/namespaces/${namespaceId}`);
        return response.data;
    } catch (error) {
        console.error(`Error deleting namespace with ID ${namespaceId}:`, error);
        throw error;
    }
}

export async function deleteUserFromNamespaceByUserId(namespaceId, userId) {
    try {
        const response = await api.delete(`/namespaces/${namespaceId}/users/${userId}`);
        return response.data;
    } catch (error) {
        console.error(`Error deleting user with ID ${userId} from namespace with ID ${namespaceId}:`, error);
        throw error;
    }
}
