<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('deployedApplications.title')" :subheading="$t('deployedApplications.subheading')" />
        <BaseDataTable v-model="search" :headers="deployedApplicationsHeaders" :items="deployedApplications" :loading="loading"
            :loadingText="$t('deployedApplications.table.loadingText')"
            :noDataText="$t('deployedApplications.table.noDataText')"
            :itemsPerPageText="$t('deployedApplications.table.itemsPerPageText')"
            :titleToolbar="$t('deployedApplications.table.title')"
            :footerMessage="$t('deployedApplications.table.footerMessage', { src: 'Stolos Custom Resources' })"
            :actionsButtonForTable="actionsButtonForTable" rowClickable
            @click:row="(event, item) => showViewDetailsDeployedAppDialog(item.item)">
        </BaseDataTable>
        <CreateDeployedAppDialog v-model="dialogDeployNewApp" @deployedAppCreated="fetchDeployedApps" />
        <ViewDetailsDeployedAppDialog v-model="dialogViewDetailsDeployedApp" :deployedApp="selectedDeployedApp" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTemplates } from "@/services/templates.service";
import CreateDeployedAppDialog from "./dialogs/CreateDeployedAppDialog.vue";
import ViewDetailsDeployedAppDialog from "./dialogs/ViewDetailsDeployedAppDialog.vue";

const { t } = useI18n();

// State
const loading = ref(false);
const search = ref('');
const dialogDeployNewApp = ref(false);
const dialogViewDetailsDeployedApp = ref(false);
const deployedApplications = ref([]);
const selectedDeployedApp = ref(null);

// Computed
const deployedApplicationsHeaders = computed(() => [
    { title: t('deployedApplications.table.headers.instanceName'), value: 'instanceName' },
    { title: t('deployedApplications.table.headers.namespace'), value: 'namespace', sortable: false, align: 'center' },
    { title: t('deployedApplications.table.headers.health'), value: 'health', sortable: false, align: 'center' },
    { title: t('deployedApplications.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }, //TODO : A voir ici si cest les bons noms de propriétés
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-text-box",
        tooltip: t('actionButtons.viewDocs'),
        text: t('actionButtons.viewDocs'),
        click: () => console.log('Docs button clicked')
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
    getTemplates()
        .then((response) => {        
            deployedApplications.value = response;
        }).catch(error => {
            console.error("Error fetching templates:", error);
        });
}
</script>