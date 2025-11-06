<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.dialogs.title')" :subheading="$t('ui.dialogs.subheading')" />
        <v-row>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.dialogs.baseDialogs.title')" :code="baseDialogTemplateCode">
                    <BaseButton :text="$t('ui.dialogs.baseDialogs.button.open')" @click="openBaseDialog" />
                    <BaseDialog v-model="isBaseDialogOpen" :title="$t('ui.dialogs.baseDialogs.title')" closable>
                        <v-form v-model="isValidForm">
                            <BaseTextfield :Textfield="formFields.name" />
                        </v-form>
                        <template #actions>
                            <BaseButton variant="outlined" :text="$t('actionButtons.cancel')"
                                @click="closeBaseDialog" />
                            <BaseButton :text="$t('actionButtons.add')" :disabled="!isValidForm"
                                @click="closeBaseDialog" />
                        </template>
                    </BaseDialog>
                </ComponentPreview>
            </v-col>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.dialogs.confirmDialogs.title')" :code="confirmDialogTemplateCode">
                    <BaseButton :text="$t('ui.dialogs.confirmDialogs.button.open')" @click="openConfirmDialog" />
                    <BaseConfirmDialog ref="confirmDialog" />
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { TextField } from '@/models/TextField';

const { t } = useI18n();
const { textfieldRules } = FormValidationRules();

// States
const isValidForm = ref(false);
const isBaseDialogOpen = ref(false);
const confirmDialog = ref(null);

const formFields = reactive({
    name: new TextField({
        label: t('ui.dialogs.baseDialogs.nameLabel'),
        required: true,
        rules: textfieldRules
    }),
});

// Computed
const baseDialogTemplateCode = computed(() => 
    `
    <BaseDialog v-model="isBaseDialogOpen" title="Test" closable>
        <v-form v-model="isValidForm">
            <BaseTextfield :Textfield="formFields.name" />
        </v-form>
        <template #actions>
            <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeBaseDialog" />
            <BaseButton :text="$t('actionButtons.add')" :disabled="!isValidForm" @click="closeBaseDialog" />
        </template>
    </BaseDialog>
    `
);
const confirmDialogTemplateCode = computed(() => 
    `
    //Template
    <BaseConfirmDialog ref="confirmDialog" />

    //Methods
    function openConfirmDialog() {
        confirmDialog.value.open({
            title: "Test Confirm Dialog",
            message: "Are you sure you want to proceed with this action?",
            confirmText: t('actionButtons.confirm'),
            onConfirm: () => {
                closeConfirmDialog();
            }
        })
    }
    `
);

// Methods
function openBaseDialog() {
    isBaseDialogOpen.value = true;
}
function closeBaseDialog() {
    isBaseDialogOpen.value = false;
}
function openConfirmDialog() {
    confirmDialog.value.open({
        title: t('ui.dialogs.confirmDialogs.title'),
        message: t('ui.dialogs.confirmDialogs.message'),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            closeConfirmDialog();
        }
    })
}
function closeConfirmDialog() {
    confirmDialog.value.close();
}
</script>
