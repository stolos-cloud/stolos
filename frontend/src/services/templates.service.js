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

export async function validateTemplate(id, instance_name, namespace, cr) {
  try {
    const response = await api.post(`/templates/${id}/validate/${instance_name}?namespace=${namespace}`, cr);
    return response.data;
  } catch (error) {
    console.error('Error validating templates:', error);
    throw error;
  }
}

export async function applyTemplate(id, instance_name, namespace, cr) {
  try {
    const response = await api.post(`/templates/${id}/apply/${instance_name}?namespace=${namespace}`, cr);
    return response.data;
  } catch (error) {
    console.error('Error validating templates:', error);
    throw error;
  }
}

export async function listDeployments(template, namespace) {
  try {
    const response = await api.get(`/deployments/list?namespace=${namespace}&template=${template}`);
    return response.data;
  } catch (error) {
    console.error('Error fetching deployemnts:', error);
    throw error;
  }
}

export async function listMyDeployments(template, namespace) {
  try {
    const response = await api.get(`/deployments/list?onlyMine=true&namespace=${namespace}&template=${template}`);
    return response.data;
  } catch (error) {
    console.error('Error fetching deployemnts:', error);
    throw error;
  }
}

export async function getDeployment(template, namespace, deployment) {
  try {
    const response = await api.get(`/deployments/get?namespace=${namespace}&template=${template}&deployment=${deployment}`);
    return response.data;
  } catch (error) {
    console.error('Error fetching deployemnt:', error);
    throw error;
  }
}


export async function deleteDeployment(template, namespace, deployment) {
  try {
    return await api.post(`/deployments/delete?namespace=${namespace}&template=${template}&deployment=${deployment}`);
  } catch (error) {
    console.error('Error deleting deployemnt:', error);
    throw error;
  }
}
