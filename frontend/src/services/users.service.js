import api from './api'

export async function getUsers() {
    try {
        const response = await api.get('/api/users');
        return response.data;
    } catch (error) {
        console.error('Error fetching users:', error);
        throw error;
    }
}

export async function createNewUser(userData) {
    try {
        const response = await api.post('/api/users/create', userData);
        return response.data;
    } catch (error) {
        console.error('Error creating user:', error);
        throw error;
    }
}

export async function getUserById(userId) {
    try {
        const response = await api.get(`/api/users/${userId}`);
        return response.data;
    } catch (error) {
        console.error(`Error fetching user with ID ${userId}:`, error);
        throw error;
    }
}

export async function updateUserRole(userId, newRole) {
    try {
        const response = await api.put(`/api/users/${userId}/role`, { role: newRole });
        return response.data;
    } catch (error) {
        console.error(`Error updating role for user with ID ${userId}:`, error);
        throw error;
    }
}

export async function deleteUserById(userId) {
    try {
        const response = await api.delete(`/api/users/${userId}`);
        return response.data;
    } catch (error) {
        console.error(`Error deleting user with ID ${userId}:`, error);
        throw error;
    }
}