<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
        />
        <v-sheet class="mt-4 border rounded">
            <v-data-table
                :headers="nodeHeaders"
                :items="nodesCloud"
                :items-length="nodesCloud.length"
                :loading=loading
                :loading-text="$t('cloudProvider.table.loadingText')"
                :no-data-text="$t('cloudProvider.table.noDataText')"
                :items-per-page="10"
                :items-per-page-text="$t('cloudProvider.table.itemsPerPageText')"
                mobile-breakpoint="md"
                :hide-default-footer="nodesCloud.length < 10"
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
            </v-data-table>
        </v-sheet>
        <BaseDialog v-model="dialogConfigurateCloudConfig" :title="$t('cloudProvider.dialogs.configurateCloudConfig.title')" width="600" closable>
          <v-form v-model="isValidConfigurateForm">
            <BaseTextfield :Textfield="formFields.region" />
            <BaseFileInput :FileInput="formFields.serviceAccountFile" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('cloudProvider.buttons.cancel')" @click="cancelConfiguration" />
            <BaseButton size="small" :text="$t('cloudProvider.buttons.updateConfiguration')" :disabled="!isValidConfigurateForm" @click="updateCloudConfiguration('add')" />
          </template>
        </BaseDialog>
        <BaseDialog v-model="dialogUpdateCloudConfig" :title="$t('cloudProvider.dialogs.updateCloudConfiguration.title')" width="600" closable>
          <v-form v-model="isValidUpdateForm">
            <BaseTextfield :Textfield="formFields.region" />
            <BaseFileInput :FileInput="formFields.serviceAccountFile" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('cloudProvider.buttons.cancel')" @click="cancelUpdateConfiguration" />
            <BaseButton size="small" :text="$t('cloudProvider.buttons.updateConfiguration')" :disabled="!isValidUpdateForm" @click="updateCloudConfiguration('update')" />
          </template>
        </BaseDialog>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import DOMPurify from "dompurify";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { FileInput } from "@/models/FileInput.js";
import { ref, reactive, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getGCPStatus, configureGCPServiceAccountUpload } from '@/services/provisioning.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

// State
const dialogConfigurateCloudConfig = ref(false);
const dialogUpdateCloudConfig = ref(false);
const isValidConfigurateForm = ref(false);
const isValidUpdateForm = ref(false);
const loading = ref(false);
const nodesCloud = ref([]);
const copiedItem = ref(null);

// Form state
const formFields = reactive({
    region: new TextField({
        label: t('cloudProvider.formfields.region'),
        type: "text",
        required: true,
        rules: textfieldRules
    }),
    serviceAccountFile: new FileInput({
        label: t('cloudProvider.formfields.serviceAccountFile'),
        required: true,
        accept: ".json, application/json",
        rules: textfieldRules
    })
});

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
    getGCPStatus().then(response => {
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

function edit(item) {
    formFields.region.value = item.region; 
    dialogUpdateCloudConfig.value = true;
}

function showDialogConfigurateCloudConfig() {
    dialogConfigurateCloudConfig.value = true;
}

function cancelConfiguration() {
    formFields.region.value = undefined;
    formFields.serviceAccountFile.value = undefined;
    dialogConfigurateCloudConfig.value = false;
}

function cancelUpdateConfiguration() {
    formFields.region.value = undefined;
    formFields.serviceAccountFile.value = undefined;
    dialogUpdateCloudConfig.value = false;
}

function updateCloudConfiguration(typeConfig) {
    if (!isValidUpdateForm.value) return;
    showOverlay();

    dialogUpdateCloudConfig.value = false;
    const payload = {
        region: DOMPurify.sanitize(formFields.region.value),
        serviceAccountFile: formFields.serviceAccountFile.value
    }
    
    configureGCPServiceAccountUpload(payload)
    .then(() => {
        typeConfig === 'add' ? 
            showNotification(t('cloudProvider.notifications.addSuccess'), 'success') : 
            showNotification(t('cloudProvider.notifications.updateSuccess'), 'success');
        fetchGCPStatus();
    })
    .catch(error => {
        console.error("Error configuring GCP Service Account:", error);
    })
    .finally(() => {
        formFields.region.value = undefined;
        formFields.serviceAccountFile.value = undefined;
        hideOverlay();
    });
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