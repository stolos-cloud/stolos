<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('dashboard.title')"
            :subheading="$t('dashboard.subheading')"
            :actions="actions"
        />
        <component :is="dashboardComponent" />
        <DownloadISOOnPremDialog v-model="dialogDownloadISOOnPremise" />
    </PortalLayout>
</template>

<script setup>
import DownloadISOOnPremDialog from '@/pages/operator/dialogs/download/DownloadISOOnPremDialog.vue';
import OperatorDashboard from '@/pages/operator/OperatorDashboard.vue';
import DeveloperDashboard from '@/pages/developer/DeveloperDashboard.vue';
import { computed, ref, onMounted } from 'vue';
//import { getClusterInfo } from '@/services/cluster.service';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';

const { t } = useI18n();
const store = useStore();

// State
const dialogDownloadISOOnPremise = ref(false);
const clusterInfo = ref(null);

// Computed
const userRole = computed(() => store.getters['user/getRole']);
const dashboardComponent = computed(() => {
    return userRole.value === 'admin' ? OperatorDashboard : DeveloperDashboard;
});
const actions = computed(() => {
    if (userRole.value === 'admin') {
        return [
            { icon: "mdi-github", text: t('actionButtons.viewRepo'), tooltip: t('actionButtons.viewRepo'), onClick: () => redirectToRepository() },
            { icon: "mdi-download", text: t('actionButtons.downloadISOOnPremise'), tooltip: t('actionButtons.downloadISOOnPremise'), onClick: () => showDownloadISODialog() }
        ];
    } else {
        return [];
    }
});

// Mounted
onMounted(() => {
    if(userRole.value === 'admin') {
        // getClusterInfo().then((response) => {
        //     if(response.gitops_configured) {
        //         clusterInfo.value = response;
        //     }
        // }).catch(error => {
        //     console.error('Failed to fetch cluster info:', error);
        // });
    }
});

// Methods
function showDownloadISODialog() {
    dialogDownloadISOOnPremise.value = true;
}
function redirectToRepository() {
    const repoLink = `https://github.com/${clusterInfo.value.gitops_repo_owner}/${clusterInfo.value.gitops_repo_name}/tree/${clusterInfo.value.gitops_branch}`;
    window.open(repoLink, '_blank');
}
</script>
