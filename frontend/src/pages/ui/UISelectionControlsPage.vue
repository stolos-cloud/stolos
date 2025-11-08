<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.selectionControls.title')" :subheading="$t('ui.selectionControls.subheading')" />
        <v-row>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.selectionControls.radioButtons.title')" :code="radioButtonsTemplateCode">
                    <BaseRadioButtons :RadioGroup="formFields.options" />
                </ComponentPreview>
            </v-col>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.selectionControls.checkboxes.title')" :code="checkboxesTemplateCode">
                    <BaseCheckbox :Checkbox="formFields.test1" />
                    <BaseCheckbox :Checkbox="formFields.test2" />
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>

import { RadioGroup } from '@/models/RadioGroup'
import { Checkbox } from '@/models/Checkbox'
import { reactive } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const radioButtonsTemplateCode = `<BaseRadioButtons :RadioButtons="formFields.options" />`;
const checkboxesTemplateCode = `<BaseCheckbox :Checkbox="formFields.test1" />
<BaseCheckbox :Checkbox="formFields.test2" />`;

// Reactive form fields
const formFields = reactive({
    options: new RadioGroup({
        label: t('ui.selectionControls.radioButtons.label'),
        options: [
            { label: t('ui.selectionControls.radioButtons.options.option1'), value: 'option1' },
            { label: t('ui.selectionControls.radioButtons.options.option2'), value: 'option2' },
            { label: t('ui.selectionControls.radioButtons.options.option3'), value: 'option3' },
        ],
    }),
    test1: new Checkbox({
        label: t('ui.selectionControls.checkboxes.options.test1')
    }),
    test2: new Checkbox({   
        label: t('ui.selectionControls.checkboxes.options.test2')
    }),
});
</script>