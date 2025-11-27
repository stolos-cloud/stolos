<template>
    <div class="view-details-node-dialog">
        <BaseDialog v-model="isOpen" :title="$t('dashboard.provision.dialogs.viewDetailsNode.title')" closable>
            <BaseCard>
                <p><b>{{ $t('dashboard.provision.dialogs.viewDetailsNode.subheading') }}</b></p>
                <v-row class="my-2">
                    <v-col v-for="field in nodeFields" :key="field.key" cols="12" md="12"
                        class="d-flex flex-column flex-sm-row justify-space-between py-1">
                        <span class="text-grey">{{ $t(field.label) }} :</span>
                        <span class="text-truncate">{{ formatFieldValue(field.key) }}</span>
                    </v-col>
                </v-row>
            </BaseCard>
            <v-divider class="my-4"></v-divider>
            <BaseCard>
                <p><b>{{ $t('dashboard.provision.dialogs.viewDetailsNode.nodeInfo.labels') }}</b></p>
                <div class="d-flex flex-wrap align-center my-2">
                    <BaseChip
                        v-for="(label, index) in (node?.labels || [])"
                        :key="index"
                        :color="getLabelColor(label)"
                    >
                        {{ label }}
                    </BaseChip>
                    <span v-if="!node?.labels?.length" class="text-grey">{{ $t('dashboard.provision.dialogs.viewDetailsNode.nodeInfo.noLabels') }}</span>
                </div>
            </BaseCard>
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
