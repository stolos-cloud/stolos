<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('provisioning.cloud.title')"
            :subheading="$t('provisioning.cloud.subheading')"
        />
        <!-- Simple test form for WebSocket provisioning -->
        <v-card class="pa-1 my-4 border">
            <v-card-title>GCP Node Provisioning Test</v-card-title>
            <v-card-text>
                <v-form v-model="isValidForm">
                    <BaseTextfield :Textfield="formFields.namePrefix" />
                    <BaseTextfield :Textfield="formFields.number" />
                    <BaseSelect v-model="formFields.role.value" :Select="formFields.role" />
                    <BaseAutoComplete :AutoComplete="formFields.zone" />
                    <BaseAutoComplete :AutoComplete="formFields.machineType" />
                    <BaseTextfield :Textfield="formFields.diskSizeGb" />
                    <BaseSelect v-model="formFields.diskType.value" :Select="formFields.diskType" />
                </v-form>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <BaseButton
                    :text="isProvisioning ? 'Provisioning...' : 'Provision Nodes'"
                    :disabled="isProvisioning || !isValidForm"
                    @click="submitProvisionRequest"
                />
            </v-card-actions>
        </v-card>

        <!-- Notebook-style sequential layout -->
        <div v-if="provisioningPhase !== 'idle'">
                <!-- Phase 1: Terraform Plan -->
                <v-card class="mb-4">
                    <v-card-title class="bg-blue-darken-2">
                        <v-icon left>mdi-file-document-outline</v-icon>
                        Terraform Plan
                        <v-chip
                            v-if="provisioningPhase === 'plan'"
                            class="ml-2"
                            color="blue"
                            size="small"
                        >
                            Running...
                        </v-chip>
                        <v-chip
                            v-else
                            class="ml-2"
                            color="success"
                            size="small"
                        >
                            Complete
                        </v-chip>
                        <v-spacer></v-spacer>
                        <v-btn
                            v-if="provisioningPhase !== 'idle' && provisioningPhase !== 'plan'"
                            color="primary"
                            variant="outlined"
                            size="small"
                            @click="downloadPlan"
                        >
                            <v-icon left>mdi-download</v-icon>
                            Download Plan
                        </v-btn>
                    </v-card-title>
                    <v-card-text>
                        <v-sheet
                            color="grey-darken-4"
                            class="pa-4"
                            rounded
                            style="max-height: 400px; overflow-y: auto"
                        >
                            <div
                                v-for="(log, index) in planLogs"
                                :key="'plan-' + index"
                                class="text-caption font-monospace mb-1"
                                :class="getLogColor(log.type)"
                            >
                                [{{ new Date(log.timestamp).toLocaleTimeString() }}] {{ log.message }}
                            </div>
                            <div v-if="planLogs.length === 0" class="text-caption text-grey">
                                Waiting for plan to start...
                            </div>
                        </v-sheet>
                    </v-card-text>
                </v-card>

                <!-- Phase 2: Approval Section (inline) -->
                <v-card v-if="provisioningPhase === 'awaiting_approval'" class="mb-4">
                    <v-card-title class="bg-orange-darken-2">
                        <v-icon left>mdi-alert-circle</v-icon>
                        Approval Required
                    </v-card-title>
                    <v-card-subtitle class="pt-2">
                        {{ approvalSummary }}
                    </v-card-subtitle>
                    <v-card-text>
                        <div class="text-body-2 mb-4">
                            Review the plan above and approve to continue with terraform apply.
                        </div>
                    </v-card-text>
                    <v-card-actions class="pa-4">
                        <v-spacer></v-spacer>
                        <v-btn
                            color="error"
                            variant="outlined"
                            size="large"
                            @click="rejectProvisioning"
                        >
                            <v-icon left>mdi-close</v-icon>
                            Reject
                        </v-btn>
                        <v-btn
                            color="success"
                            variant="flat"
                            size="large"
                            @click="approveProvisioning"
                        >
                            <v-icon left>mdi-check</v-icon>
                            Approve & Apply
                        </v-btn>
                    </v-card-actions>
                </v-card>

                <!-- Phase 3: Terraform Apply -->
                <v-card v-if="['apply', 'complete', 'error'].includes(provisioningPhase)" class="mb-4">
                    <v-card-title class="bg-green-darken-2">
                        <v-icon left>mdi-play-circle-outline</v-icon>
                        Terraform Apply
                        <v-chip
                            v-if="provisioningPhase === 'apply'"
                            class="ml-2"
                            color="blue"
                            size="small"
                        >
                            Running...
                        </v-chip>
                        <v-chip
                            v-else-if="provisioningPhase === 'complete'"
                            class="ml-2"
                            color="success"
                            size="small"
                        >
                            Complete
                        </v-chip>
                        <v-chip
                            v-else-if="provisioningPhase === 'error'"
                            class="ml-2"
                            color="error"
                            size="small"
                        >
                            Error
                        </v-chip>
                    </v-card-title>
                    <v-card-text>
                        <v-sheet
                            color="grey-darken-4"
                            class="pa-4"
                            rounded
                            style="max-height: 400px; overflow-y: auto"
                        >
                            <div
                                v-for="(log, index) in applyLogs"
                                :key="'apply-' + index"
                                class="text-caption font-monospace mb-1"
                                :class="getLogColor(log.type)"
                            >
                                [{{ new Date(log.timestamp).toLocaleTimeString() }}] {{ log.message }}
                            </div>
                            <div v-if="applyLogs.length === 0" class="text-caption text-grey">
                                Waiting for apply to start...
                            </div>
                        </v-sheet>
                    </v-card-text>
                </v-card>
        </div>

        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular indeterminate></v-progress-circular>
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue';
import api from '@/services/api';
import { StorageService } from '@/services/storage.service';
import { TextField } from '@/models/TextField.js';
import { Select } from '@/models/Select.js';
import { AutoComplete } from '@/models/AutoComplete';
import { FormValidationRules } from '@/composables/FormValidationRules.js';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';

const { t } = useI18n();
const store = useStore();
const { textfieldRules } = FormValidationRules();

const overlay = ref(false);
const isProvisioning = ref(false);
const isValidForm = ref(false);
const planLogs = ref([]);
const applyLogs = ref([]);
const status = ref('');
const ws = ref(null);
const provisioningPhase = ref('idle'); // idle, plan, awaiting_approval, apply, complete, error
const approvalSummary = ref('');
const currentRequestId = ref('');

const form = ref({
    name_prefix: 'gcp-worker',
    number: 1,
    zone: 'us-central1-a',
    machine_type: 'n1-standard-2',
    role: 'worker',
    disk_size_gb: 100,
    disk_type: 'pd-standard',
});

// Computed
const roleProvisioningTypes = computed(() => store.getters['referenceLists/getProvisioningRoles']);
const cloudZones = computed(() => store.getters['referenceLists/getCloudZones']);
const availableMachineTypes = computed(() => {
    const zone = formFields.zone.value;
    if (!zone) return undefined;
    const machines =  store.getters['referenceLists/getMachinesTypesByZone'](zone);
    return machines.map(machine => ({
        label: `${machine.name} - ${machine.description}`,
        value: machine.name
    }));
});
const diskTypes = computed(() => [
    { label: 'Standard Persistent Disk (pd-standard)', value: 'pd-standard' },
    { label: 'Balanced Persistent Disk (pd-balanced)', value: 'pd-balanced' },
    { label: 'SSD Persistent Disk (pd-ssd)', value: 'pd-ssd' },
    { label: 'Extreme Persistent Disk (pd-extreme)', value: 'pd-extreme' },
]);

// Mounted
onMounted(() => {
    formFields.namePrefix.value = form.value.name_prefix;
    formFields.number.value = form.value.number;
    formFields.zone.value = form.value.zone;
    formFields.machineType.value = form.value.machine_type;
    formFields.role.value = form.value.role;
    formFields.diskSizeGb.value = form.value.disk_size_gb;
    formFields.diskType.value = form.value.disk_type;
});

// Form state
const formFields = reactive({
    namePrefix: new TextField({
        label: t('provisioning.cloud.nodeFormfields.namePrefix'),
        type: "text",
        required: true,
        rules: textfieldRules
    }),
    number: new TextField({
        label: t('provisioning.cloud.nodeFormfields.numberOfNodes'),
        type: "number",
        min: 1,
        max: 10,
        required: true,
        rules: textfieldRules
    }),
    role: new Select({
        label: t('provisioning.cloud.nodeFormfields.role'),
        options: roleProvisioningTypes.value,
        required: true,
        rules: textfieldRules
    }),
    zone: new AutoComplete({
        label: t('provisioning.cloud.nodeFormfields.zone'),
        items: cloudZones,
        required: true,
        rules: textfieldRules
    }),
    machineType: new AutoComplete({
        label: t('provisioning.cloud.nodeFormfields.machineType'),
        items: availableMachineTypes,
        required: true,
        disabled: computed(() => !formFields.zone.value),
        rules: textfieldRules
    }),
    diskSizeGb: new TextField({
        label: t('provisioning.cloud.nodeFormfields.diskSizeGb'),
        type: "number",
        min: 10,
        max: 65536,
        required: true,
        rules: textfieldRules
    }),
    diskType: new Select({
        label: t('provisioning.cloud.nodeFormfields.diskType'),
        options: diskTypes.value,
        required: true,
        rules: textfieldRules
    })
});

// Watch form fields and sync back to form
watch(() => formFields.namePrefix.value, (newVal) => { form.value.name_prefix = newVal; });
watch(() => formFields.number.value, (newVal) => { form.value.number = parseInt(newVal) || 1; });
watch(() => formFields.zone.value, (newVal) => { form.value.zone = newVal; });
watch(() => formFields.machineType.value, (newVal) => { form.value.machine_type = newVal; });
watch(() => formFields.role.value, (newVal) => { form.value.role = newVal; });
watch(() => formFields.diskSizeGb.value, (newVal) => { form.value.disk_size_gb = parseInt(newVal) || 100; });
watch(() => formFields.diskType.value, (newVal) => { form.value.disk_type = newVal; });

const getLogColor = type => {
    switch (type) {
        case 'error':
            return 'text-red';
        case 'status':
            return 'text-blue';
        case 'log':
            return 'text-green-lighten-2';
        default:
            return 'text-white';
    }
};

const submitProvisionRequest = async () => {
    isProvisioning.value = true;
    planLogs.value = [];
    applyLogs.value = [];
    provisioningPhase.value = 'plan';
    status.value = 'Sending request...';

    try {
        // Step 1: POST to create provision request
        const response = await api.post('/api/gcp/nodes/provision', form.value);
        const requestId = response.data.request_id;
        currentRequestId.value = requestId;

        status.value = 'Connecting to stream...';
        planLogs.value.push({
            type: 'log',
            message: `Request ID: ${requestId}`,
            timestamp: new Date(),
        });

        // Step 2: Connect to WebSocket
        connectWebSocket(requestId);
    } catch (error) {
        isProvisioning.value = false;
        provisioningPhase.value = 'error';
        planLogs.value.push({
            type: 'error',
            message: `Error: ${error.response?.data?.error || error.message}`,
            timestamp: new Date(),
        });
    }
};

const connectWebSocket = requestId => {
    // Get WebSocket URL (convert http to ws)
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = StorageService.get('token');
    const wsUrl = `${wsProtocol}//${window.location.host}/api/gcp/nodes/provision/${requestId}/stream?token=${token}`;

    planLogs.value.push({
        type: 'log',
        message: `Connecting to WebSocket...`,
        timestamp: new Date(),
    });

    ws.value = new WebSocket(wsUrl);

    ws.value.onopen = () => {
        status.value = 'Connected';
        planLogs.value.push({
            type: 'status',
            message: 'WebSocket connected!',
            timestamp: new Date(),
        });
    };

    ws.value.onmessage = event => {
        try {
            const message = JSON.parse(event.data);

            if (message.type === 'log') {
                // Route logs to apply section if we're applying or completed
                // (completion logs should stay in apply section)
                const targetLogs = (provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete')
                    ? applyLogs
                    : planLogs;

                targetLogs.value.push({
                    type: 'log',
                    message: message.payload.message,
                    timestamp: new Date(),
                });
            } else if (message.type === 'status') {
                status.value = message.payload.status;

                // Determine which logs section to add status message to BEFORE updating phase
                const targetLogs = (provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete')
                    ? applyLogs
                    : planLogs;

                // Update phase based on status
                if (message.payload.status === 'planning') {
                    provisioningPhase.value = 'plan';
                } else if (message.payload.status === 'awaiting_approval') {
                    provisioningPhase.value = 'awaiting_approval';
                } else if (message.payload.status === 'applying') {
                    provisioningPhase.value = 'apply';
                } else if (message.payload.status === 'completed') {
                    provisioningPhase.value = 'complete';
                }

                targetLogs.value.push({
                    type: 'status',
                    message: `Status: ${message.payload.status}`,
                    timestamp: new Date(),
                });
            } else if (message.type === 'plan') {
                // Plan output - add to plan logs
                planLogs.value.push({
                    type: 'log',
                    message: message.payload.plan,
                    timestamp: new Date(),
                });
            } else if (message.type === 'approval_required') {
                approvalSummary.value = message.payload.summary;
                provisioningPhase.value = 'awaiting_approval';
                planLogs.value.push({
                    type: 'status',
                    message: 'Waiting for approval...',
                    timestamp: new Date(),
                });
            } else if (message.type === 'complete') {
                applyLogs.value.push({
                    type: 'status',
                    message: 'Provisioning completed!',
                    timestamp: new Date(),
                });
                provisioningPhase.value = 'complete';
                isProvisioning.value = false;
                ws.value.close();
            } else if (message.type === 'error') {
                // Route error to apply logs if we're in apply phase, otherwise to plan logs
                const targetLogs = (provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete')
                    ? applyLogs
                    : planLogs;

                targetLogs.value.push({
                    type: 'error',
                    message: `Error: ${message.payload.error}`,
                    timestamp: new Date(),
                });
                provisioningPhase.value = 'error';
                isProvisioning.value = false;
                ws.value.close();
            }
        } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
        }
    };

    ws.value.onerror = error => {
        // Route to apply logs if we're in apply/complete phase
        const targetLogs = (provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete')
            ? applyLogs
            : planLogs;

        targetLogs.value.push({
            type: 'error',
            message: `WebSocket error: ${error}`,
            timestamp: new Date(),
        });
        provisioningPhase.value = 'error';
        isProvisioning.value = false;
    };

    ws.value.onclose = () => {
        // Route to apply logs if we're in apply/complete phase
        const targetLogs = (provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete')
            ? applyLogs
            : planLogs;

        targetLogs.value.push({
            type: 'status',
            message: 'WebSocket connection closed',
            timestamp: new Date(),
        });
        isProvisioning.value = false;
    };
};

const approveProvisioning = () => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
        ws.value.send(JSON.stringify({ action: 'approve' }));
        provisioningPhase.value = 'apply';
        applyLogs.value.push({
            type: 'status',
            message: 'Provisioning approved by user',
            timestamp: new Date(),
        });
    }
};

const rejectProvisioning = () => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
        ws.value.send(JSON.stringify({ action: 'reject', reason: 'Rejected by user' }));
        provisioningPhase.value = 'error';
        applyLogs.value.push({
            type: 'error',
            message: 'Provisioning rejected by user',
            timestamp: new Date(),
        });
        isProvisioning.value = false;
    }
};

const downloadPlan = async () => {
    try {
        const token = StorageService.get('token');
        const url = `/api/gcp/nodes/provision/${currentRequestId.value}/plan`;

        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to download plan');
        }

        const blob = await response.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = downloadUrl;
        link.download = `plan-${currentRequestId.value}.txt`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(downloadUrl);
    } catch (error) {
        console.error('Failed to download plan:', error);
    }
};
</script>
