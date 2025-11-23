<template>
    <div class="create-template-dialog">
        <BaseDialog v-model="isOpen" :title="$t('templateDefinitions.dialogs.createTemplate.title')">
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
import { ref, reactive, watch, computed, onMounted } from "vue";
import { useStore } from 'vuex';
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { Select } from "@/models/Select.js";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import { getScaffolds } from "@/services/scaffolds.service";
import { createNewTemplate } from "@/services/templates.service";
import DOMPurify from "dompurify";


const { t } = useI18n();
const store = useStore();
const { textfieldRules, textfieldSlugRules } = FormValidationRules();
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

// Computed
const scaffoldsList = computed(() => store.getters['referenceLists/getScaffolds']);

// Mounted
onMounted(async () => {
    if(scaffoldsList.value.length === 0) {
        await getScaffolds().then(scaffolds => {
            store.dispatch('referenceLists/setScaffolds', scaffolds);
        });
    }
});

// Form state
const formFields = reactive({
    templateName: new TextField({
        label: computed(() => t('templateDefinitions.formfields.templateName')),
        type: "text",
        required: true,
        rules: textfieldSlugRules
    }),
    scaffold: new Select({
        label: computed(() => t('templateDefinitions.formfields.scaffold')),
        options: scaffoldsList,
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
    showOverlay();
    
    const templateName = DOMPurify.sanitize(formFields.templateName.value);
    createNewTemplate({ templateName, scaffoldName: formFields.scaffold.value })
        .then(() => {
            showNotification(t('templateDefinitions.notifications.createSuccess', { templateName }), 'success');
            emit('templateCreated');
        })
        .catch((error) => {
            showNotification(t('templateDefinitions.notifications.createError'), 'error');
            console.error("Error creating team:", error);
        })
        .finally(() => {
            closeDialog();
            hideOverlay();
        });
}
</script>