<template>
    <BaseDialog v-model="isOpen" :title="$t('cloudProvider.dialogs.configurateCloudConfig.title')" closable>
        <v-form v-model="isValidForm">
            <BaseTextfield :Textfield="formFields.region" />
            <BaseFileInput :FileInput="formFields.serviceAccountFile" />
        </v-form>
        <template #actions>
            <BaseButton variant="outlined" :text="$t('cloudProvider.buttons.cancel')" @click="closeDialog" />
            <BaseButton :text="$t('cloudProvider.buttons.updateConfiguration')" :disabled="!isValidForm" @click="addCloudConfiguration" />
        </template>
    </BaseDialog>
</template>

<script setup>
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { FileInput } from "@/models/FileInput.js";
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { configureGCPServiceAccountUpload } from '@/services/provisioning.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import DOMPurify from "dompurify";

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    }
});

// State
const isValidForm = ref(false);
const isOpen = ref(props.modelValue);

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

// Emits
const emit = defineEmits(['update:modelValue', 'cloudConfigurationAdded']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.region.value = undefined;
    formFields.serviceAccountFile.value = undefined;
    isOpen.value = false;
}
function addCloudConfiguration() {
    if (!isValidForm.value) return;
    showOverlay();

    const payload = {
        region: DOMPurify.sanitize(formFields.region.value),
        serviceAccountFile: formFields.serviceAccountFile.value
    }
    
    configureGCPServiceAccountUpload(payload)
        .then(() => {
            showNotification(t('cloudProvider.notifications.addSuccess'), 'success')
            emit('cloudConfigurationAdded');
        })
        .catch(error => {
            console.error("Error configuring GCP Service Account:", error);
        })
        .finally(() => {
            closeDialog();
            hideOverlay();
        });
}
</script>