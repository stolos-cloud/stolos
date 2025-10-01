import api from './api'

/*-------------------------------------
    On-Premises Service Functions
-------------------------------------*/
export async function downloadISO(payload) {
    try {
        const response = await api.post('/api/isos/generate', payload, {
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

export async function getConnectedNodes({status}) {
    try {
        const response = await api.get('/api/nodes', { params: { status: status } });
        return response.data;
    } catch (error) {
        console.error('Error fetching connected nodes:', error);
        throw error;
    }
}

export async function createNodesWithRoleAndLabels({ nodes }) {
    try {
        const response = await api.post('/api/nodes', { params: { nodes } });
        return response.data;
    } catch (error) {
        console.error('Error creating connected nodes:', error);
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

export async function configureGCPServiceAccount(payload) {
    try {
        const response = await api.put('/api/gcp/service-account', payload);
        return response.data;
    } catch (error) {
        console.error('Error configuring GCP service account:', error);
        throw error;
    }
}