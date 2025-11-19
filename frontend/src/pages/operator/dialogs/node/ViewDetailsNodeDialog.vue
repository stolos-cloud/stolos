<template>
    <div class="view-details-node-dialog">
        <BaseDialog v-model="isOpen" :title="$t('dashboard.provision.dialogs.viewDetailsNode.title')" closable>
            <BaseTitle :level="3" :title="$t('dashboard.provision.dialogs.viewDetailsNode.subheading')" />
            <v-row class="mt-4">
                <v-col
                    v-for="field in nodeFields"
                    :key="field.key"
                    cols="12"
                    sm="6"
                    md="4"
                >
                    <div class="d-flex flex-column">
                        <strong class="mb-1">{{ $t(field.label) }}:</strong>
                        <span
                            :class="field.key === 'id' ? 'text-mono' : ''"
                            :style="field.key === 'id' ? 'word-break: break-all; user-select: all;' : ''"
                        >
                            {{ formatFieldValue(field.key) }}
                        </span>
                    </div>
                </v-col>
            </v-row>

            <!-- Labels section - full width at bottom -->
            <v-divider class="my-4"></v-divider>
            <div class="mt-4">
                <strong class="mb-2 d-block">{{ $t('dashboard.provision.dialogs.viewDetailsNode.nodeInfo.labels') }}:</strong>
                <div class="d-flex flex-wrap align-center">
                    <v-chip
                        v-for="(label, idx) in (node?.labels || [])"
                        :key="idx"
                        :color="getLabelColor(label)"
                        class="ma-1"
                        label
                    >
                        {{ label }}
                    </v-chip>
                    <span v-if="!node?.labels?.length" class="text-grey">{{ $t('dashboard.provision.dialogs.viewDetailsNode.nodeInfo.noLabels') }}</span>
                </div>
            </div>

            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.close')" @click="closeDialog" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { LabelColorHandler } from "@/composables/LabelColorHandler";

const { getLabelColor } = LabelColorHandler();

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

// Computed fields based on provider
const nodeFields = computed(() => {
    const provider = props.node?.provider?.toLowerCase();
    const isOnPrem = provider === 'onprem';
    const isCloud = provider && provider !== 'onprem';

    const fields = [
        { key: 'id', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.id' },
        { key: 'ip_address', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.ipAddress' },
    ];

    // Show MAC address for on-prem nodes
    if (isOnPrem) {
        fields.push({ key: 'mac_address', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.macAddress' });
    }

    // Show Instance ID for cloud providers
    if (isCloud) {
        fields.push({ key: 'instance_id', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.instanceId' });
    }

    fields.push(
        { key: 'architecture', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.architecture' },
        { key: 'created_at', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.createdAt' },
        { key: 'updated_at', label: 'dashboard.provision.dialogs.viewDetailsNode.nodeInfo.updatedAt' }
    );

    return fields;
});

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    emit('update:modelValue', false);
}

function formatFieldValue(key) {
    const value = props.node?.[key];
    if (!value) return '-';

    if (key === 'created_at' || key === 'updated_at') {
        return formatDateShort(value);
    }

    return value;
}

function formatDateShort(dateString) {
    if (!dateString) return '';
    const options = { year: 'numeric', month: '2-digit', day: '2-digit' };
    return new Date(dateString).toLocaleDateString(undefined, options).split('/').reverse().join('-');
}
</script>
