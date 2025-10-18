import api from './api';

/*-------------------------------------
    On-Premises Service Functions
-------------------------------------*/
export async function generateISO(payload) {
    try {
        const response = await api.post('/api/iso/generate', payload);
        return response.data; // Returns { download_url }
    } catch (error) {
        console.error('Error generating ISO:', error);
        throw error;
    }
}

export async function getConnectedNodes({ status } = {}) {
    try {
        const params = {};
        if (status !== undefined) params.status = status;

        const response = await api.get('/api/nodes', { params });
        return response.data;
    } catch (error) {
        console.error('Error fetching connected nodes:', error);
        throw error;
    }
}

export async function provisionNodes({ nodes }) {
    try {
        const response = await api.post('/api/nodes/provision', { nodes });
        return response.data;
    } catch (error) {
        console.error('Error provisioning nodes:', error);
        throw error;
    }
}

export async function updateNodesLabels({ nodes }) {
    try {
        const response = await api.put('/api/nodes/config', { nodes });
        return response.data;
    } catch (error) {
        console.error('Error updating node labels:', error);
        throw error;
    }
}

//For testing purposes only - to be removed in production
export async function createSamplesNodes() {
    try {
        const response = await api.post('/api/nodes/samples');
        return response.data;
    } catch (error) {
        console.error('Error creating samples nodes with pending status:', error);
        throw error;
    }
}

/*-------------------------------------
    Cloud Service Functions
-------------------------------------*/
export async function getGCPStatus() {
    try {
        const response = await api.get('/api/gcp/status');
        return response.data;
    } catch (error) {
        console.error('Error fetching GCP status:', error);
        throw error;
    }
}

export async function configureGCPServiceAccountUpload({ region, serviceAccountFile }) {
    try {
        const formData = new FormData();
        formData.append('region', region);
        formData.append('service_account_file', serviceAccountFile);
        const response = await api.post('/api/gcp/configure/upload', formData);
        return response.data;
    } catch (error) {
        console.error('Error configuring GCP service account:', error);
        throw error;
    }
}

export async function getAvailableGCPResources() {
    try {
        const response = await api.get('/api/gcp/resources');
        return response.data;
    } catch (error) {
        console.error('Error fetching GCP resources:', error);
        throw error;
    }
}
