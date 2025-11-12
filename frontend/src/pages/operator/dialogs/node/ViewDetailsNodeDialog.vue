<template>
    <div class="view-details-node-dialog">
        <BaseDialog v-model="isOpen" :title="$t('dashboard.provision.dialogs.viewDetailsNode.title')" closable>
            <BaseTitle :level="3" :title="$t('dashboard.provision.dialogs.viewDetailsNode.subheading')" />
            <v-row class="mt-4">
                <v-col cols="12" sm="6" v-for="field in nodeFields" :key="field.key">
                    <div class="d-flex justify-space-between">
                        <strong>{{ $t(field.label) }}:</strong>
                        <span>
                            {{ field.key === 'created_at' || field.key === 'updated_at' ? formatDateShort(node?.[field.key]) : node?.[field.key] }}
                        </span>
                    </div>
                </v-col>
            </v-row>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, watch } from "vue";

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    node: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);
const nodeFields = ref([
    { key: 'ip_address', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.ipAddress' },
    { key: 'mac_address', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.macAddress' },
    { key: 'instance_id', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.instanceId' },
    { key: 'architecture', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.architecture' },
    { key: 'created_at', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.createdAt' },
    { key: 'updated_at', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.updatedAt' },
])

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function formatDateShort(dateString) {
    if (!dateString) return '';
    const options = { year: 'numeric', month: '2-digit', day: '2-digit' };
    return new Date(dateString).toLocaleDateString(undefined, options).split('/').reverse().join('-');
}
</script>