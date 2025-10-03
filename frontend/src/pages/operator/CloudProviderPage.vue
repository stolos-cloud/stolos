<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
        />
        <div class="mt-4">
            <h3>{{ $t('cloudProvider.table.title') }}</h3>
            <v-data-table
                :headers="nodeHeaders"
                :items="nodesCloud"
                :items-length="nodesCloud.length"
                :loading=loading
                :loading-text="$t('cloudProvider.table.loadingText')"
                :no-data-text="$t('cloudProvider.table.noDataText')"
                :items-per-page="10"
                :items-per-page-text="$t('cloudProvider.table.itemsPerPageText')"
                class="elevation-8 mt-2"
                mobile-breakpoint="md"
            >
              <template v-slot:item.actions="{ item }">
                <v-icon color="medium-emphasis" icon="mdi-pencil" size="small" @click="edit(item)"></v-icon>
              </template>
            </v-data-table>
        </div>
        <BaseDialog v-model="dialogConfigurateCloudConfig" :title="$t('cloudProvider.dialogs.configurateCloudConfig.title')" width="600" closable>
          <v-form v-model="isValidConfigurateForm">
            <BaseTextfield :Textfield="formFields.region" />
            <BaseFileInput :FileInput="formFields.serviceAccountFile" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('cloudProvider.buttons.cancel')" @click="cancelConfiguration" />
            <BaseButton size="small" :text="$t('cloudProvider.buttons.updateConfiguration')" :disabled="!isValidConfigurateForm" @click="updateCloudConfiguration" />
          </template>
        </BaseDialog>
        <BaseDialog v-model="dialogUpdateCloudConfig" :title="$t('cloudProvider.dialogs.updateCloudConfiguration.title')" width="600" closable>
          <v-form v-model="isValidUpdateForm">
            <BaseTextfield :Textfield="formFields.region" />
            <BaseFileInput :FileInput="formFields.serviceAccountFile" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('cloudProvider.buttons.cancel')" @click="cancelUpdateConfiguration" />
            <BaseButton size="small" :text="$t('cloudProvider.buttons.updateConfiguration')" :disabled="!isValidUpdateForm" @click="updateCloudConfiguration" />
          </template>
        </BaseDialog>
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular
              indeterminate
            />
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { FileInput } from "@/models/FileInput.js";
import { ref, reactive, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getGCPStatus, configureGCPServiceAccountUpload } from '@/services/provisioning.service';

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();

// State
const dialogConfigurateCloudConfig = ref(false);
const dialogUpdateCloudConfig = ref(false);
const isValidConfigurateForm = ref(false);
const isValidUpdateForm = ref(false);
const loading = ref(false);
const overlay = ref(false);
const nodesCloud = ref([]);

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

//Methods
function fetchGCPStatus() {
  getGCPStatus().then(response => {
    if(response.configured) {      
      nodesCloud.value = [response.config];
    }
  }).catch(error => {
    console.error("Error fetching GCP status:", error);
  });
}

function edit(item) {
  formFields.region.value = item.region; 
  dialogUpdateCloudConfig.value = true;
}

function cancelConfiguration() {
    formFields.region.value = undefined;
    dialogConfigurateCloudConfig.value = false;
}

function cancelUpdateConfiguration() {
    formFields.region.value = undefined;
    dialogUpdateCloudConfig.value = false;
}

function updateCloudConfiguration() {
  if (!isValidUpdateForm.value) return;

  overlay.value = true;
  dialogUpdateCloudConfig.value = false;
  const payload = {
    region: formFields.region.value,
    serviceAccountFile: formFields.serviceAccountFile.value
  }
  
  configureGCPServiceAccountUpload(payload)
  .then(() => {
    fetchGCPStatus();
  })
  .catch(error => {
    console.error("Error configuring GCP Service Account:", error);
  })
  .finally(() => {
    overlay.value = false;
    formFields.projectId.value = undefined;
    formFields.region.value = undefined;
    formFields.serviceAccountFile.value = undefined;
  });
}

</script>

<style scoped>
.chip-input .v-field__input {
  padding: 0 8px !important;
}
</style>