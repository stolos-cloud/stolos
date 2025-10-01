<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
            :actions="actions"
        />
        <v-form v-model="isValid">
            <BaseTextfield :Textfield="textfields.projectId" />
            <BaseTextfield :Textfield="textfields.region" />
        </v-form>
        <BaseButton :text="$t('cloudProvider.buttons.configure')" class="mt-4" :disabled="!isValid" @click="configureCloudServiceAccount(textfields.projectId.value, textfields.region.value)" />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import BaseTextfield from "@/components/base/BaseTextfield.vue";
import { TextField } from "@/models/TextField.js";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { ref, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { getGCPStatus, configureGCPServiceAccount } from '@/services/provisioning.service';

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();

// State
const isValid = ref(false);

// Form state
const textfields = reactive({
  projectId: new TextField({
    label: t('cloudProvider.projectId'),
    type: "text",
    required: true,
    rules: textfieldRules
  }),
  region: new TextField({
    label: t('cloudProvider.region'),
    type: "text",
    required: true,
    rules: textfieldRules
  }),
});

//Mounted
onMounted(() => {
  fetchGCPStatus();
});

//Methods
function fetchGCPStatus() {
  getGCPStatus().then(response => {
    console.log(response);
  }).catch(error => {
    console.error("Error fetching GCP status:", error);
  });
}

function configureCloudServiceAccount(projectId, region) {
  configureGCPServiceAccount({ projectId, region })
  .then(response => {
    console.log("GCP Service Account configured successfully:", response);
  }).catch(error => {
    console.error("Error configuring GCP Service Account:", error);
  });
}

</script>
