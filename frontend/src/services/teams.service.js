import api from './api';

export async function getTeams() {
    try {
        const response = await api.get('/api/teams');
        return response.data;
    } catch (error) {
        console.error('Error fetching teams:', error);
        throw error;
    }
}

export async function createNewTeam(teamData) {
    try {
        const response = await api.post('/api/teams', teamData);
        return response.data;
    } catch (error) {
        console.error('Error creating team:', error);
        throw error;
    }
}

export async function getTeamDetails(teamId) {
    try {
        const response = await api.get(`/api/teams/${teamId}`);
        return response.data;
    } catch (error) {
        console.error(`Error fetching team with ID ${teamId}:`, error);
        throw error;
    }
}

export async function addUserIdToTeam(teamId, user) {
    try {
        const response = await api.post(`/api/teams/${teamId}/users`, user);
        return response.data;
    } catch (error) {
        console.error(`Error adding user to team with ID ${teamId}:`, error);
        throw error;
    }
}

export async function deleteTeamById(teamId) {
    try {
        const response = await api.delete(`/api/teams/${teamId}`);
        return response.data;
    } catch (error) {
        console.error(`Error deleting team with ID ${teamId}:`, error);
        throw error;
    }
}

export async function deleteUserFromTeamByUserId(teamId, userId) {
    try {
        const response = await api.delete(`/api/teams/${teamId}/users/${userId}`);
        return response.data;
    } catch (error) {
        console.error(`Error deleting user with ID ${userId} from team with ID ${teamId}:`, error);
        throw error;
    }
}
