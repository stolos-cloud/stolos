<template>
    <BaseDialog v-model="isOpen" :title="$t('administration.teams.dialogs.createTeam.title')" closable>
        <v-form v-model="isValidForm">
            <BaseTextfield :Textfield="formFields.teamName" />
        </v-form>
        <template #actions>
            <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
            <BaseButton  :text="$t('actionButtons.create')" :disabled="!isValidForm" @click="createTeam" />
        </template>
    </BaseDialog>
</template>

<script setup>
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { createNewTeam } from "@/services/teams.service";

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();

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
    teamName: new TextField({
        label: t('administration.teams.formfields.teamName'),
        type: "text",
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'teamCreated', 'loading']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.teamName.value = undefined;
    emit('update:modelValue', false);
}
function createTeam() {
    if(!isValidForm.value) return;
    emit('loading', true);

    const teamName = formFields.teamName.value;
    createNewTeam({ name: teamName })
        .then(() => {
            emit('teamCreated');
        })
        .catch((error) => {
            console.error("Error creating team:", error);
        })
        .finally(() => {
            closeDialog();
            emit('loading', false);
        });
}
</script>