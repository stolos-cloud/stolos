<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.alerts.title')" :subheading="$t('ui.alerts.subheading')" />
        <v-row>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.alerts.notice.title')" :code="textfieldCode">
                    <BaseTextfield :Textfield="fields.textfield" :iconAction="showIconTextfield" />
                    <template #controls>
                        <BaseSelect v-model="fields.textfieldType.value" :Select="fields.textfieldType" />
                        <BaseCheckbox :Checkbox="fields.readonly" />
                        <BaseCheckbox :Checkbox="fields.icon" />
                    </template>
                </ComponentPreview>
            </v-col>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.inputs.autocompleteInput.title')" :code="autoCompleteCode">
                    <BaseAutoComplete :AutoComplete="fields.autocomplete" />
                    <template #controls>
                        <BaseCheckbox :Checkbox="fields.autocompleteDisabled" />
                        <BaseCheckbox :Checkbox="fields.autocompleteMultiple" />
                    </template>
                </ComponentPreview>
            </v-col>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.inputs.fileInput.title')" :code="fileInputCode">
                    <BaseFileInput :FileInput="fields.fileInput" />
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>
import { Checkbox } from '@/models/Checkbox'
import { TextField } from '@/models/TextField';
import { AutoComplete } from '@/models/AutoComplete';
import { Select } from '@/models/Select'
import { FileInput } from '@/models/FileInput';
import { computed, onMounted, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import { FormValidationRules } from "@/composables/FormValidationRules.js";

const { t } = useI18n();
const { textfieldRules, emailRules } = FormValidationRules();

// Computed
const textfieldCode = computed(() =>
    `
    //Template
    <BaseTextfield :Textfield="fields.textfield" ${fields.icon.value ? `:iconAction="'mdi-eye'"` : ''} />

    //Reactive
    textfield: new TextField({
        label: t('ui.alerts.notice.controls.textfield.label'),
        placeholder: t('ui.alerts.notice.controls.textfield.placeholder'),
        type: ${fields.textfieldType.value},
        readonly: ${fields.readonly.value},
        required: true,
        rules: textfieldRules
    }),
    `
);
const autoCompleteCode = computed(() =>
    `
    //Template
    <BaseAutoComplete :AutoComplete="fields.autocomplete" />

    //Reactive
    autocomplete: new AutoComplete({
        label: t('provisioning.cloud.nodeFormfields.zone'),
        items: [
            { label: 'test1', value: 'test1' },
            { label: 'test2', value: 'test2' },
            { label: 'test3', value: 'test3' },
            { label: 'test4', value: 'test4' }
        ],
        disabled: ${fields.autocompleteDisabled.value},
        multiple: ${fields.autocompleteMultiple.value},
        required: true,
        rules: textfieldRules
    }),
    `
);
const fileInputCode = computed(() =>
    `
    //Template
    <BaseFileInput :FileInput="fields.fileInput" />

    //Reactive
    fileInput: new FileInput({
        label: t('ui.inputs.fileInput.file.label'),
        required: true,
        accept: ".json, application/json",
        rules: textfieldRules
    }),
    `
);
const showIconTextfield = computed(() => fields.icon.value ? 'mdi-eye' : undefined);

// Reactive form fields
const fields = reactive({
    textfield: new TextField({
        label: t('ui.inputs.textfieldInput.label'),
        type: computed(() => fields.textfieldType.value),
        readonly: computed(() => fields.readonly.value),
        required: true,
        rules: computed(() => fields.textfieldType.value === 'email' ? emailRules : textfieldRules)
    }),
    textfieldType: new Select({
        label: t('ui.inputs.textfieldInput.type.label'),
        options: [
            { label: t('ui.inputs.textfieldInput.type.options.text'), value: 'text' },
            { label: t('ui.inputs.textfieldInput.type.options.number'), value: 'number' },
            { label: t('ui.inputs.textfieldInput.type.options.date'), value: 'date' },
            { label: t('ui.inputs.textfieldInput.type.options.email'), value: 'email' },
            { label: t('ui.inputs.textfieldInput.type.options.password'), value: 'password' }
        ],
    }),
    readonly: new Checkbox({
        label: t('ui.inputs.textfieldInput.checkboxes.readonly')
    }),
    icon: new Checkbox({
        label: t('ui.inputs.textfieldInput.checkboxes.withIcon')
    }),
    autocomplete: new AutoComplete({
        label: t('ui.inputs.autocompleteInput.label'),
        items: [
            { label: 'test1', value: 'test1' },
            { label: 'test2', value: 'test2' },
            { label: 'test3', value: 'test3' },
            { label: 'test4', value: 'test4' }
        ],
        disabled: computed(() => fields.autocompleteDisabled.value),
        multiple: computed(() => fields.autocompleteMultiple.value),
        required: true,
        rules: textfieldRules
    }),
    autocompleteDisabled: new Checkbox({
        label: t('ui.inputs.autocompleteInput.checkboxes.disabled')
    }),
    autocompleteMultiple: new Checkbox({
        label: t('ui.inputs.autocompleteInput.checkboxes.multiple')
    }),
    fileInput: new FileInput({
        label: t('ui.inputs.fileInput.file.label'),
        required: true,
        accept: ".json, application/json",
        rules: textfieldRules
    }),
});

// Mounted
onMounted(() => {
    fields.textfieldType.value = 'text';
});
</script>