<template>
    <div class="create-template-dialog">
        <BaseDialog v-model="isOpen" :title="$t('templateDefinitions.dialogs.createTemplate.title')" closable>
            <v-form v-model="isValidForm">
                <BaseTextfield :Textfield="formFields.templateName" />
                <BaseSelect v-model="formFields.scaffold.value" :Select="formFields.scaffold" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton :text="$t('actionButtons.create')" :disabled="!isValidForm" @click="createTemplate" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { Select } from "@/models/Select.js";
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
    templateName: new TextField({
        label: t('templateDefinitions.formfields.templateName'),
        type: "text",
        required: true,
        rules: textfieldRules
    }),
    scaffold: new Select({
        label: t('templateDefinitions.formfields.scaffold'),
        options: [{ label: 'Yes', value: true }, { label: 'No', value: false }], // TODO : Pour le moment en attente si il faut une liste speciales
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'templateCreated']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.templateName.value = undefined;
    formFields.scaffold.value = undefined;
    emit('update:modelValue', false);
}
function createTemplate() {
    if (!isValidForm.value) return;
    // showOverlay();

    // createNewTemplate({ name: formFields.templateName.value, scaffold: formFields.scaffold.value })
    //     .then((response) => {
    //         const sanitizeRepoLink = DOMPurify.sanitize(response.repoLink);
    //         showNotification(t('templateDefinitions.notifications.createSuccess', { repoLink: sanitizeRepoLink }), 'success', true);
    //         emit('templateCreated');
    //     })
    //     .catch((error) => {
    //         showNotification(t('templateDefinitions.notifications.createError'), 'error');
    //         console.error("Error creating team:", error);
    //     })
    //     .finally(() => {
    //         closeDialog();
    //         hideOverlay();
    //     });
}
</script>