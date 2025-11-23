<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
        />
        <BaseDataTable
            :headers="nodeHeaders"
            :items="nodesCloud"
            :loading="loading"
            :loadingText="$t('cloudProvider.table.loadingText')"
            :noDataText="$t('cloudProvider.table.noDataText')"
            :itemsPerPageText="$t('cloudProvider.table.itemsPerPageText')"
            :titleToolbar="$t('cloudProvider.table.title')"
            :actionsButtonForTable="actionsButtonForTable"
        >
            <template v-slot:top>
                <BaseToolbarTable :title="$t('cloudProvider.table.title')" :buttons="actionsButtonForTable" />
            </template>
            <template v-slot:[`item.service_account_email`]="{ item }">
                <div class="d-flex align-center">
                    <span class="d-none d-md-inline text-truncate">{{ item.service_account_email }}</span>
                    <span class="d-md-none">{{ item.service_account_email }}</span>
                    <v-btn
                        class="ml-1"
                        :icon="copiedItem === item.service_account_email ? 'mdi-check' : 'mdi-content-copy'"
                        size="x-small"
                        variant="text"
                        @click="copyToClipboard(item.service_account_email)"
                    />
                </div>
            </template>
            <template v-slot:[`item.actions`]="{ item }">
                <v-icon color="medium-emphasis" icon="mdi-pencil" size="small" @click="edit(item)"></v-icon>
            </template>
        </BaseDataTable>
        <ConfigurateCloudDialog v-model="dialogConfigurateCloudConfig" @cloudConfigurationAdded="fetchGCPStatus" />
        <UpdateCloudDialog v-model="dialogUpdateCloudConfig" :region="selectedRegion" @cloudConfigurationUpdated="fetchGCPStatus" />        
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getGCPStatus } from '@/services/provisioning.service';
import ConfigurateCloudDialog from "@/pages/dialogs/cloud/ConfigurateCloudDialog.vue";
import UpdateCloudDialog from "@/pages/dialogs/cloud/UpdateCloudDialog.vue";

const { t } = useI18n();

// State
const dialogConfigurateCloudConfig = ref(false);
const dialogUpdateCloudConfig = ref(false);
const loading = ref(false);
const nodesCloud = ref([]);
const copiedItem = ref(null);
const selectedRegion = ref("");

//Mounted
onMounted(() => {
    fetchGCPStatus();
});

// Computed
const nodeHeaders = computed(() => [
    { title: t('cloudProvider.table.headers.projectId'), value: 'project_id'},
    { title: t('cloudProvider.table.headers.bucketName'), value: 'bucket_name' },
    { title: t('cloudProvider.table.headers.region'), value: 'region' },
    { title: t('cloudProvider.table.headers.emailSA'), value: 'service_account_email', width: "30%" },
    { title: t('cloudProvider.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);
const isValidAddConfig = computed(() => nodesCloud.value.length === 0);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchGCPStatus
    },
    {
        icon: "mdi-plus",
        tooltip: t('cloudProvider.buttons.addNewConfiguration'),
        text: t('cloudProvider.buttons.addNewConfiguration'),
        disabled: !isValidAddConfig.value,
        click: showDialogConfigurateCloudConfig
    }
]);

//Methods
function fetchGCPStatus() {
    loading.value = true;

    getGCPStatus()
        .then(response => {
            if(response.gcp?.configured) {
                nodesCloud.value = [response.gcp];
            } else {
                nodesCloud.value = [];
            }
        }).catch(error => {
            console.error("Error fetching GCP status:", error);
            nodesCloud.value = [];
        }).finally(() => {
            loading.value = false;
        });
}
function showDialogConfigurateCloudConfig() {
    dialogConfigurateCloudConfig.value = true;
}
function edit(item) {
    selectedRegion.value = item.region; 
    dialogUpdateCloudConfig.value = true;
}
function copyToClipboard(value) {
    navigator.clipboard.writeText(value).then(() => {
        copiedItem.value = value;
        setTimeout(() => {
            copiedItem.value = null;
        }, 2000);
    })
}
</script>