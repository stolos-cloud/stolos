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

export async function validateTemplate(id, instance_name, team, cr) {
  try {
    const response = await api.post(`/templates/${id}/validate/${instance_name}?team=${team}`, cr);
    return response.data;
  } catch (error) {
    console.error('Error validating templates:', error);
    throw error;
  }
}

export async function applyTemplate(id, instance_name, team, cr) {
  try {
    const response = await api.post(`/templates/${id}/apply/${instance_name}?team=${team}`, cr);
    return response.data;
  } catch (error) {
    console.error('Error validating templates:', error);
    throw error;
  }
}
