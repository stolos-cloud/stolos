import api from './api';

export async function getTemplates() {
    try {
        const response = await api.get('/templates');
        return response.data;
    } catch (error) {
        console.error('Error fetching templates:', error);
        throw error;
    }
}

export async function getTemplate(id) {
    try {
        const response = await api.get(`/templates/${id}`);
        return response.data;
    } catch (error) {
        console.error('Error fetching templates:', error);
        throw error;
    }
}

export async function validateTemplate({ id, instance_name, namespace, cr }) {
    try {
        const response = await api.post(
            `/templates/${id}/validate/${instance_name}?namespace=${namespace}`,
            cr
        );
        return response.data;
    } catch (error) {
        console.error('Error validating templates:', error);
        throw error;
    }
}

export async function applyTemplate({ id, instance_name, namespace, cr }) {
    try {
        const response = await api.post(
            `/templates/${id}/apply/${instance_name}?namespace=${namespace}`,
            cr
        );
        return response.data;
    } catch (error) {
        console.error('Error applying templates:', error);
        throw error;
    }
}

export async function createNewTemplate({ scaffoldName, templateName }) {
    try {
        const response = await api.post('/templates/create', null, {
            params: { scaffoldName, templateName },
        });
        return response.data;
    } catch (error) {
        console.error('Error creating new template:', error);
        throw error;
    }
}
