<template>
    <div class="create-namespace-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.namespaces.dialogs.createNamespace.title')" closable>
            <v-form v-model="isValidForm">
                <BaseTextfield :Textfield="formFields.namespaceName" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton  :text="$t('actionButtons.create')" :disabled="!isValidForm" @click="createNamespace" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { createNewNamespace } from "@/services/namespaces.service";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

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
    namespaceName: new TextField({
        label: t('administration.namespaces.formfields.namespaceName'),
        type: "text",
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'namespaceCreated']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.namespaceName.value = undefined;
    emit('update:modelValue', false);
}
function createNamespace() {
    if(!isValidForm.value) return;
    showOverlay();

    const namespaceName = formFields.namespaceName.value;
    createNewNamespace({ name: namespaceName })
        .then(() => {
            showNotification(t('administration.namespaces.notifications.createNamespaceSuccess'), 'success');
            emit('namespaceCreated');
        })
        .catch((error) => {
            console.error("Error creating namespace:", error);
        })
        .finally(() => {
            closeDialog();
            hideOverlay();
        });
}
</script>
