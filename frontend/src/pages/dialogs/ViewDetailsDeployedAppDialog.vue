<template>
    <div class="view-details-template-dialog">
        <BaseDialog v-model="isOpen" :title="$t('deployedApplications.dialogs.viewDetailsDeployedApp.title', { appName: deploymentName })" closable>
            <BaseCard>
                <p><b>{{ $t('deployedApplications.dialogs.viewDetailsDeployedApp.subheading') }}</b></p>
                <v-row class="my-2">
                    <v-col v-for="field in deployFields" :key="field.key" cols="12" md="12"
                        class="d-flex flex-column flex-sm-row justify-space-between py-1">
                        <span class="text-grey">{{ $t(field.label) }} :</span>
                        <span class="text-truncate">{{ formatFieldValue(field.key) }}</span>
                    </v-col>
                </v-row>
            </BaseCard>
            <BaseExpansion :title="$t('deployedApplications.dialogs.viewDetailsDeployedApp.yamlDeployment')">
                <v-sheet color="grey-darken-4 pa-1" rounded style="position: relative; overflow-y: auto; max-height: 400px;">
                    <v-btn
                        v-if="yamlText"
                        :icon="copiedItem === yamlText ? 'mdi-check' : 'mdi-content-copy'"
                        size="x-small"
                        variant="text"
                        style="position: absolute; top: 6px; right: 6px;"
                        @click="copyToClipboard(yamlText)"
                    />
                    <pre style="white-space: pre-wrap;">{{ yamlText }}</pre>
                </v-sheet>
            </BaseExpansion>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import yaml from "js-yaml";
import { getDeployment } from "@/services/templates.service";

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
        
    },
    deployment: {
        required: true,
    }
});

const deployFields = [
  { key: 'image', label: 'deployedApplications.dialogs.viewDetailsDeployedApp.details.image' },
  { key: 'replicas', label: 'deployedApplications.dialogs.viewDetailsDeployedApp.details.replicas' },
  { key: 'port', label: 'deployedApplications.dialogs.viewDetailsDeployedApp.details.port' },
  { key: 'nodePort', label: 'deployedApplications.dialogs.viewDetailsDeployedApp.details.nodePort' },
  { key: 'owner', label: 'deployedApplications.dialogs.viewDetailsDeployedApp.details.owner' },
];

// State
const isOpen = ref(props.modelValue);
const deploymentDetails = ref(null);
const deploymentName = ref('');
const copiedItem = ref(null);

// Computed
const yamlText = computed(() => {
    if (!deploymentDetails.value) return '';
    const obj = JSON.parse(JSON.stringify(deploymentDetails.value));

    return yaml.dump(obj, { 
        noRefs: true,
        lineWidth: -1
    });
});

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if (val && props.deployment) {
        getDeploymentDetails(props.deployment);
    }
});

// Methods
function getDeploymentDetails(deployment) {    
    getDeployment({
        template: deployment.template,
        namespace: deployment.namespace,
        deployment: deployment.name
    })
        .then((response) => {            
            deploymentName.value = response.metadata?.name || '';
            deploymentDetails.value = response;
        }).catch((error) => {
            console.error("Error fetching deployment details:", error);
            deploymentDetails.value = null;
        });
}
function formatFieldValue(key) {
    const dep = deploymentDetails.value || props.deployment;
    switch (key) {
        case 'image':
            return dep.spec?.image || '-';
        case 'replicas':
            return dep.spec?.replicas ?? '-';
        case 'port':
            return dep.spec?.port ?? '-';
        case 'nodePort':
            return dep.spec?.nodePort ?? '-';
        case 'owner':
            return dep.metadata?.ownerReferences?.[0]?.name || '-';
        default:
            return '-';
    }
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
</script>