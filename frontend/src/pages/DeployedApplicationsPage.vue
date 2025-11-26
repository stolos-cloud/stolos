<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('deployedApplications.title')" :subheading="$t('deployedApplications.subheading')" />
        <BaseDataTable v-model="search" :headers="deployedApplicationsHeaders" :items="deployedApplications"
            :loading="loading" :loadingText="$t('deployedApplications.table.loadingText')"
            :noDataText="$t('deployedApplications.table.noDataText')"
            :itemsPerPageText="$t('deployedApplications.table.itemsPerPageText')"
            :titleToolbar="$t('deployedApplications.table.title')"
            :footerMessage="$t('deployedApplications.table.footerMessage', { src: 'Stolos Custom Resources' })"
            :actionsButtonForTable="actionsButtonForTable" rowClickable
            @click:row="(event, item) => showViewDetailsDeployedAppDialog(item.item)">

            <template #[`item.healthy`]="{ item }">
                <BaseChip :color="getStatusInfo(item).color">
                    <template #prepend> 
                        <v-icon>{{ getStatusInfo(item).icon }}</v-icon>
                    </template>
                    {{ getStatusInfo(item).text }}
                </BaseChip>
            </template>
            <template #[`item.actions`]="{ item }">
                <v-btn 
                    v-tooltip="{ text: $t('deployedApplications.buttons.deleteDeployment') }" 
                    icon="mdi-delete" size="small" variant="plain" 
                    @click="deleteDeploymentDialog(item)" 
                />
            </template>
        </BaseDataTable>
        <CreateDeployedAppDialog v-model="dialogDeployNewApp" @deployedAppCreated="fetchDeployedApps" />
        <ViewDetailsDeployedAppDialog v-model="dialogViewDetailsDeployedApp" :deployedApp="selectedDeployedApp" />
        <BaseConfirmDialog ref="confirmDialog" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { listDeployments, listMyDeployments, deleteDeployment } from "@/services/templates.service";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import CreateDeployedAppDialog from "@/pages/dialogs/CreateDeployedAppDialog.vue";
import ViewDetailsDeployedAppDialog from "@/pages/dialogs/ViewDetailsDeployedAppDialog.vue";

const { t } = useI18n();
const store = useStore();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

// State
const loading = ref(false);
const search = ref('');
const dialogDeployNewApp = ref(false);
const dialogViewDetailsDeployedApp = ref(false);
const deployedApplications = ref([]);
const selectedDeployedApp = ref(null);
const confirmDialog = ref(null);

// Computed
const isAdmin = computed(() => store.getters['user/getRole'] === 'admin');
const deployedApplicationsHeaders = computed(() => [
    { title: t('deployedApplications.table.headers.instanceName'), value: 'name', width: '25%' },
    { title: t('deployedApplications.table.headers.template'), value: 'template', align: 'center' },
    { title: t('deployedApplications.table.headers.namespace'), value: 'namespace', align: 'center' },
    { title: t('deployedApplications.table.headers.health'), value: 'healthy', align: 'center' },
    { title: t('deployedApplications.table.headers.actions'), value: 'actions', align: 'center' },
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-text-box",
        tooltip: t('actionButtons.viewDocs'),
        text: t('actionButtons.viewDocs'),
        click: redirectToWikiDocs
    },
    {
        icon: "mdi-plus",
        tooltip: t('deployedApplications.buttons.deployNewApp'),
        text: t('deployedApplications.buttons.deployNewApp'),
        click: showDeployNewAppDialog
    }
]);

//Mounted
onMounted(() => {
    fetchDeployedApps();
});

// Methods
function showDeployNewAppDialog() {
    dialogDeployNewApp.value = true;
}
function showViewDetailsDeployedAppDialog(deployedApp) {
    selectedDeployedApp.value = deployedApp;
    dialogViewDetailsDeployedApp.value = true;
}
function fetchDeployedApps() {
    loading.value = true;
    const request = isAdmin.value ? listDeployments : listMyDeployments;

    request({ template: "", namespace: "" })
         .then((response) => {
            deployedApplications.value = response.map(
                deployment => toCamelCaseObject(deployment)
            );
         }).catch(error => {
             console.error("Error fetching my deployed applications:", error);
         }).finally(() => {
             loading.value = false;
         });
}
function redirectToWikiDocs() {
    window.open("https://github.com/stolos-cloud/stolos/wiki");
}
function deleteDeploymentDialog(item) {
    confirmDialog.value.open({
        title: t('deployedApplications.dialogs.deleteDeployment.title'),
        message: t('deployedApplications.dialogs.deleteDeployment.confirmationText', { appName: item.name }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteDeploymentConfirmed(item);
        }
    })
}
function deleteDeploymentConfirmed(item) {
    showOverlay();

    deleteDeployment({ template: item.template, namespace: item.namespace, deployment: item.name })
        .then(() => {
            showNotification(t('deployedApplications.notifications.deleteSuccess', { appName: item.name }), 'success');
            fetchDeployedApps();
        })
        .catch((error) => {
            console.error("Error deleting deployment:", error);
            showNotification(t('deployedApplications.notifications.deleteError'), 'error');
        })
        .finally(() => {
            hideOverlay();
        });
}
function getStatusInfo(isHealthy) {
    return {
        text: isHealthy ? t('deployedApplications.table.healthySuccess') : t('deployedApplications.table.healthyFailed'),
        icon: isHealthy ? 'mdi-check-circle-outline' : 'mdi-close-circle-outline',
        color: isHealthy ? 'success' : 'error'
    };
}
function toCamelCaseObject(obj) {
  return Object.fromEntries(
    Object.entries(obj).map(([key, value]) => [
      key.charAt(0).toLowerCase() + key.slice(1),
      value
    ])
  );
}
</script>