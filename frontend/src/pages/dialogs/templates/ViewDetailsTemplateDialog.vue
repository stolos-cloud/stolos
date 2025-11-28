<template>
    <div class="view-details-template-dialog">
        <BaseDialog v-model="isOpen"
            :title="$t('templateDefinitions.dialogs.viewDetailsTemplate.title', { template: template?.name })" closable>
            <BaseExpansion :title="$t('templateDefinitions.dialogs.viewDetailsTemplate.crdDefinition')">
                <template #actions>
                    <BaseChip class="ml-2">
                        {{ template?.version }}
                    </BaseChip>
                </template>
                <v-sheet color="grey-darken-4" rounded style="position: relative;">
                    <v-btn
                        v-if="templateDetails?.jsonSchema"
                        :icon="copiedItem === templateDetails?.jsonSchema ? 'mdi-check' : 'mdi-content-copy'"
                        size="x-small"
                        variant="text"
                        style="position: absolute; top: 6px; right: 6px;"
                        @click="copyToClipboard(templateDetails?.jsonSchema)"
                    />
                    <pre style="white-space: pre-wrap;">{{ templateDetails?.jsonSchema }}</pre>
                </v-sheet>
            </BaseExpansion>
            <BaseCard>
                <template #title>
                    <BaseTitle :level="6" :title="$t('templateDefinitions.dialogs.viewDetailsTemplate.deployedApps')" />
                    <BaseChip class="ml-2">
                        {{ $t('templateDefinitions.dialogs.viewDetailsTemplate.totalDeployedApps', { count: template?.deployedApps.length || 0 }) }}
                    </BaseChip>
                    <v-spacer />
                    <v-btn variant="text" icon="mdi-open-in-new" color="primary" size="small"
                        @click="redirectToDeployedApplications">
                    </v-btn>
                </template>
                <v-virtual-scroll :items="template.deployedApps" max-height="150" v-if="template.deployedApps.length > 0">
                    <template v-slot:default="{ item }">
                        <v-list lines="two">
                            <v-list-item :key="item.Name" :title="item.Name" class="border rounded"
                                style="background-color: rgba(var(--v-theme-list-item));">
                                <template #subtitle>
                                    <div class="d-flex align-center">
                                        {{ $t('templateDefinitions.dialogs.viewDetailsTemplate.namespace') }} : {{ item.Namespace }}
                                    </div>
                                </template>
                                <template v-slot:append>
                                    <BaseChip :color="getStatusInfo(item.Healthy).color">
                                        <template #prepend> 
                                            <v-icon>{{ getStatusInfo(item.Healthy).icon }}</v-icon>
                                        </template>
                                        {{ getStatusInfo(item.Healthy).text }}
                                    </BaseChip>
                                </template>
                            </v-list-item>
                        </v-list>
                    </template>
                </v-virtual-scroll>
            </BaseCard>
        </BaseDialog>
    </div>
</template>

<script setup>
import { getTemplate } from "@/services/templates.service";
import { ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

const router = useRouter();
const { t } = useI18n();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    template: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);
const templateDetails = ref([]);
const copiedItem = ref(null);

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if (val && props.template) {
        getTemplateDetailsFromName(props.template.name);
    }
});

// Methods
function getTemplateDetailsFromName(name) {
    getTemplate(name)
        .then((response) => {
            templateDetails.value = response;
        })
        .catch((error) => {
            console.error("Error fetching template details:", error);
        });
}
function redirectToDeployedApplications() {
    router.push('/deployed-applications');
}
function copyToClipboard(text) {
    navigator.clipboard.writeText(text)
        .then(() => {
            copiedItem.value = text;
            setTimeout(() => {
                copiedItem.value = null;
            }, 2000);
        })
}
function getStatusInfo(isHealthy) {
    return {
        text: isHealthy ? t('templateDefinitions.dialogs.viewDetailsTemplate.healthy') : t('templateDefinitions.dialogs.viewDetailsTemplate.failed'),
        icon: isHealthy ? 'mdi-check-circle-outline' : 'mdi-close-circle-outline',
        color: isHealthy ? 'success' : 'error'
    };
}
</script>

<style>
.v-expansion-panel--active>.v-expansion-panel-title:not(.v-expansion-panel-title--static) {
    border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.1) !important;
}

.v-expansion-panel-text__wrapper {
    padding: 10px 12px 10px !important;
}
</style>