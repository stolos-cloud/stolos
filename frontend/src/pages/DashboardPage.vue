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
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';

const { t } = useI18n();
const store = useStore();

// State
const dialogDownloadISOOnPremise = ref(false);

// Computed
const userRole = computed(() => store.getters['user/getRole']);
const dashboardComponent = computed(() => {
    return userRole.value === 'admin' ? OperatorDashboard : DeveloperDashboard;
});
const actions = computed(() => {
    if (userRole.value === 'admin') {
        return [
            { icon: "mdi-download", text: t('actionButtons.downloadISOOnPremise'), tooltip: t('actionButtons.downloadISOOnPremise'), onClick: () => showDownloadISODialog() }
        ];
    } else {
        return [];
    }
});

// Methods
function showDownloadISODialog() {
    dialogDownloadISOOnPremise.value = true;
}
</script>
