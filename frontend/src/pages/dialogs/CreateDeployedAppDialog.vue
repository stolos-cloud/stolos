<template>
    <div class="deployed-app-template-dialog">
        <BaseDialog v-model="isOpen" :title="$t('deployedApplications.dialogs.createDeployedApp.title')" closable>
            <v-form v-model="isValidForm">
                <BaseTextfield :Textfield="formFields.instanceName" />
                <v-row class="ga-0">
                    <v-col cols="12" lg="6" md="6" class="py-0">
                        <BaseSelect v-model="formFields.namespace.value" :Select="formFields.namespace" />
                    </v-col>
                    <v-col cols="12" lg="6" md="6" class="py-0">
                        <BaseSelect v-model="formFields.template.value" :Select="formFields.template" />
                    </v-col>
                </v-row>
                <BaseExpansion :title="$t('deployedApplications.dialogs.createDeployedApp.crdConfiguration')"
                    :disabled="!formFields.template.value">
                    <DeployTemplate :key="deployTemplateKey" ref="deployTemplateRef"
                        :templateId="formFields.template.value" />
                </BaseExpansion>
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')"
                    @click="closeCreateDeployedAppDialog" />
                <BaseButton :text="$t('actionButtons.validate')" :disabled="!isValidForm"
                    @click="validateTemplateDeployment" />
            </template>
        </BaseDialog>
        <BaseDialog v-model="isConfirmDeploymentDialogOpen"
            :title="$t('deployedApplications.dialogs.confirmDeploymentDialog.notice.title')" closable persistent>
            <BaseNotice type="success" :title="$t('deployedApplications.dialogs.confirmDeploymentDialog.notice.title')"
                :text="$t('deployedApplications.dialogs.confirmDeploymentDialog.notice.text')" />
            <BaseCard>
                <p><b>{{ $t('deployedApplications.dialogs.confirmDeploymentDialog.deploymentDetails.title') }}</b></p>
                <v-row class="my-2">
                    <v-col v-for="(item, i) in deploymentDetails" :key="i" cols="12" md="12"
                        class="d-flex flex-column flex-sm-row justify-space-between py-1">
                        <span class="text-grey">{{ item.label }} :</span>
                        <span class="text-truncate">{{ item.value }}</span>
                    </v-col>
                </v-row>
            </BaseCard>
            <template #actions>
                <BaseButton size="small" variant="outlined" :text="$t('actionButtons.close')"
                    @click="closeConfirmDeploymentDialog" />
                <BaseButton size="small"
                    :text="$t('deployedApplications.dialogs.confirmDeploymentDialog.buttons.confirmDeployment')"
                    @click="confirmDeployment" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { TextField } from "@/models/TextField.js";
import { Select } from "@/models/Select.js";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import { getTemplates, validateTemplate, applyTemplate } from "@/services/templates.service";
import { getNamespaces } from "@/services/namespaces.service";
import DOMPurify from "dompurify";
import DeployTemplate from "../developer/DeployTemplate.vue";


const { t } = useI18n();
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
const templates = ref([]);
const namespaces = ref([]);
const deployTemplateKey = ref(0);
const deployTemplateRef = ref(null);
const isConfirmDeploymentDialogOpen = ref(false);

// Computed
const templatesOptions = computed(() => templates.value);
const namespacesOptions = computed(() => namespaces.value);
const deploymentDetails = computed(() => ([
    { label: t('deployedApplications.formfields.instanceName'), value: formFields.instanceName.value },
    { label: t('deployedApplications.formfields.namespace'), value: formFields.namespace.value },
    { label: t('deployedApplications.formfields.template'), value: formFields.template.value }
]));

// Form state
const formFields = reactive({
    instanceName: new TextField({
        label: computed(() => t('deployedApplications.formfields.instanceName')),
        type: "text",
        required: true,
        rules: textfieldSlugRules
    }),
    namespace: new Select({
        label: computed(() => t('deployedApplications.formfields.namespace')),
        type: "text",
        options: namespacesOptions,
        required: true,
        rules: textfieldRules
    }),
    template: new Select({
        label: computed(() => t('deployedApplications.formfields.template')),
        type: "text",
        options: templatesOptions,
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'deployedAppCreated']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if (val) {
        getTemplatesForDeployment();
        getNamespacesForDeployment();
    }
});
watch(() => formFields.template.value, (newVal) => {
    if (!newVal) return;
    deployTemplateKey.value++;
});

// Methods
function closeCreateDeployedAppDialog() {
    formFields.instanceName.value = undefined;
    formFields.namespace.value = undefined;
    formFields.template.value = undefined;
    emit('update:modelValue', false);
}
function closeConfirmDeploymentDialog() {
    isConfirmDeploymentDialogOpen.value = false;
}
function getTemplatesForDeployment() {
    getTemplates().then((response) => {
        templates.value = response.map(template => ({
            label: template.name,
            value: template.name
        }));
    }).catch(error => {
        console.error("Error fetching templates:", error);
    });
}
function getNamespacesForDeployment() {
    getNamespaces()
        .then((response) => {
            namespaces.value = response.namespaces.map(namespace => ({
                label: namespace.name,
                value: namespace.name
            }));
        })
        .catch((error) => {
            console.error("Error fetching namespaces:", error);
        });
}
function validateTemplateDeployment() {
    if (!isValidForm.value || !deployTemplateRef.value) return;
    showOverlay();

    validateTemplate({
        id: formFields.template.value,
        instance_name: DOMPurify.sanitize(formFields.instanceName.value),
        namespace: formFields.namespace.value,
        cr: deployTemplateRef.value.getEditorContent()
    })
        .then((response) => {
            if (response.status === "ok") {
                isConfirmDeploymentDialogOpen.value = true;
            }
        })
        .catch((error) => {
            console.error("Error validating template deployment:", error);
            showNotification(t('deployedApplications.notifications.validateError'), 'error');
            closeCreateDeployedAppDialog();
        })
        .finally(() => {
            hideOverlay();
        });
}
function confirmDeployment() {
    if (!isValidForm.value) return;
    showOverlay();

    applyTemplate({
        id: formFields.template.value,
        instance_name: DOMPurify.sanitize(formFields.instanceName.value),
        namespace: formFields.namespace.value,
        cr: deployTemplateRef.value.getEditorContent()
    })
        .then((response) => {
            if (response.status === "ok") {
                showNotification(t('deployedApplications.notifications.deploySuccess'), 'success');
                emit('deployedAppCreated');
            }
        })
        .catch((error) => {
            console.error("Error creating deployed application :", error);
            showNotification(t('deployedApplications.notifications.deployError'), 'error');
        })
        .finally(() => {
            closeConfirmDeploymentDialog();
            closeCreateDeployedAppDialog();
            hideOverlay();
        });
}
</script>