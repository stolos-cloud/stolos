<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('provisioning.cloud.title')"
            :subheading="$t('provisioning.cloud.subheading')"
        />
        <v-card class="pa-1 my-4 border">
            <v-card-title>{{ $t('provisioning.cloud.cardTitle') }}</v-card-title>
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
                <v-tooltip :text="provisionButtonTooltip" location="top">
                    <template v-slot:activator="{ props }">
                        <div v-bind="props">
                            <BaseButton
                                :text="
                                    isProvisioning
                                        ? $t('provisioning.cloud.buttons.provisioning')
                                        : $t('provisioning.cloud.buttons.provision')
                                "
                                :disabled="
                                    isProvisioning ||
                                    !isValidForm ||
                                    infrastructureStatus !== 'ready'
                                "
                                @click="submitProvisionRequest"
                            />
                        </div>
                    </template>
                </v-tooltip>
            </v-card-actions>
        </v-card>

        <!-- Notebook-style sequential layout -->
        <div v-if="provisioningPhase !== 'idle'">
            <!-- Phase 1: Terraform Plan -->
            <v-card class="mb-4">
                <v-card-title class="bg-orange-darken-2">
                    <v-icon left>mdi-file-document-outline</v-icon>
                    {{ $t('provisioning.cloud.phases.terraformPlan') }}
                    <v-chip
                        v-if="provisioningPhase === 'plan'"
                        class="ml-2 text-white"
                        color="blue"
                        size="small"
                        variant="flat"
                    >
                        {{ $t('provisioning.cloud.status.running') }}
                    </v-chip>
                    <v-chip
                        v-else
                        class="ml-2 text-white"
                        color="success"
                        size="small"
                        variant="flat"
                    >
                        {{ $t('provisioning.cloud.status.complete') }}
                    </v-chip>
                    <v-spacer></v-spacer>
                    <v-btn
                        v-if="provisioningPhase !== 'idle' && provisioningPhase !== 'plan'"
                        color="grey-darken-4"
                        variant="outlined"
                        size="small"
                        @click="downloadPlan"
                    >
                        <v-icon left>mdi-download</v-icon>
                        {{ $t('provisioning.cloud.buttons.downloadPlan') }}
                    </v-btn>
                </v-card-title>
                <v-card-text>
                    <v-sheet color="grey-darken-4" class="pa-4" rounded>
                        <div
                            v-for="(log, index) in planLogs"
                            :key="'plan-' + index"
                            class="text-caption font-monospace mb-1"
                            :class="getLogColor(log.type)"
                        >
                            [{{ new Date(log.timestamp).toLocaleTimeString() }}] {{ log.message }}
                        </div>
                        <div v-if="planLogs.length === 0" class="text-caption text-grey">
                            {{ $t('provisioning.cloud.messages.waitingPlan') }}
                        </div>
                    </v-sheet>
                </v-card-text>
            </v-card>

            <!-- Plan Phase Resources -->
            <ResourceTracker
                v-if="provisioningPhase !== 'idle' && (planResources.length > 0 || workflowData)"
                :resources="planResources"
                :workflow="workflowData"
                :title="$t('provisioning.cloud.resources.plannedChanges')"
                phase="plan"
                :empty-message="$t('provisioning.cloud.resources.emptyMessages.analyzing')"
                class="mb-4"
            />

            <!-- Phase 2: Approval Section -->
            <v-card v-if="provisioningPhase === 'awaiting_approval'" class="mb-4">
                <v-card-title class="bg-orange-darken-2">
                    <v-icon left>mdi-alert-circle</v-icon>
                    {{ $t('provisioning.cloud.phases.approvalRequired') }}
                </v-card-title>
                <v-card-subtitle class="pt-2">
                    {{ approvalSummary }}
                </v-card-subtitle>
                <v-card-text>
                    <div class="text-body-2 mb-4">
                        {{ $t('provisioning.cloud.messages.reviewPlan') }}
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
                        {{ $t('provisioning.cloud.buttons.reject') }}
                    </v-btn>
                    <v-btn color="success" variant="flat" size="large" @click="approveProvisioning">
                        <v-icon left>mdi-check</v-icon>
                        {{ $t('provisioning.cloud.buttons.approve') }}
                    </v-btn>
                </v-card-actions>
            </v-card>

            <!-- Phase 3: Terraform Apply -->
            <v-card v-if="['apply', 'complete', 'error'].includes(provisioningPhase)" class="mb-4">
                <v-card-title class="bg-green-darken-2">
                    <v-icon left>mdi-play-circle-outline</v-icon>
                    {{ $t('provisioning.cloud.phases.terraformApply') }}
                    <v-chip
                        v-if="provisioningPhase === 'apply'"
                        class="ml-2 text-white"
                        color="blue"
                        size="small"
                        variant="flat"
                    >
                        {{ $t('provisioning.cloud.status.running') }}
                    </v-chip>
                    <v-chip
                        v-else-if="provisioningPhase === 'complete'"
                        class="ml-2 text-white"
                        color="success"
                        size="small"
                        variant="flat"
                    >
                        {{ $t('provisioning.cloud.status.complete') }}
                    </v-chip>
                    <v-chip
                        v-else-if="provisioningPhase === 'error'"
                        class="ml-2 text-white"
                        color="error"
                        size="small"
                        variant="flat"
                    >
                        {{ $t('provisioning.cloud.status.error') }}
                    </v-chip>
                    <v-spacer></v-spacer>
                    <v-btn
                        v-if="['complete', 'error'].includes(provisioningPhase)"
                        color="grey-darken-4"
                        variant="outlined"
                        size="small"
                        @click="downloadApply"
                    >
                        <v-icon left>mdi-download</v-icon>
                        {{ $t('provisioning.cloud.buttons.downloadApply') }}
                    </v-btn>
                </v-card-title>
                <v-card-text>
                    <v-sheet color="grey-darken-4" class="pa-4" rounded>
                        <div
                            v-for="(log, index) in applyLogs"
                            :key="'apply-' + index"
                            class="text-caption font-monospace mb-1"
                            :class="getLogColor(log.type)"
                        >
                            [{{ new Date(log.timestamp).toLocaleTimeString() }}] {{ log.message }}
                        </div>
                        <div v-if="applyLogs.length === 0" class="text-caption text-grey">
                            {{ $t('provisioning.cloud.messages.waitingApply') }}
                        </div>
                    </v-sheet>
                </v-card-text>
            </v-card>

            <!-- Apply Phase Resources -->
            <ResourceTracker
                v-if="
                    ['apply', 'complete', 'error'].includes(provisioningPhase) &&
                    (applyResources.length > 0 || workflowData)
                "
                :resources="applyResources"
                :workflow="workflowData"
                :title="$t('provisioning.cloud.resources.resourcesCreated')"
                phase="apply"
                :empty-message="$t('provisioning.cloud.resources.emptyMessages.noResources')"
                class="mb-4"
            />
        </div>

        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular indeterminate></v-progress-circular>
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import api, { WS_BASE_URL } from '@/services/api';
import { StorageService } from '@/services/storage.service';
import { TextField } from '@/models/TextField.js';
import { Select } from '@/models/Select.js';
import { AutoComplete } from '@/models/AutoComplete';
import { FormValidationRules } from '@/composables/FormValidationRules.js';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';
import ResourceTracker from '@/components/terraform/ResourceTracker.vue';

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
const eventWs = ref(null);
const provisioningPhase = ref('idle'); // idle, plan, awaiting_approval, apply, complete, error
const approvalSummary = ref('');
const currentRequestId = ref('');
const infrastructureStatus = ref('unconfigured'); // unconfigured, pending, initializing, ready, failed

// Resource tracking
const planResources = ref([]);
const applyResources = ref([]);

// Computed tooltip for provision button
const provisionButtonTooltip = computed(() => {
    if (isProvisioning.value) {
        return t('provisioning.cloud.tooltips.provisioning');
    }
    if (!isValidForm.value) {
        return t('provisioning.cloud.tooltips.invalidForm');
    }
    if (infrastructureStatus.value !== 'ready') {
        const statusKey = infrastructureStatus.value || 'unconfigured';
        return t(`provisioning.cloud.tooltips.infrastructure.${statusKey}`);
    }
    return t('provisioning.cloud.tooltips.ready');
});
const workflowData = ref(null);
const resourcesMap = ref({});

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
    const machines = store.getters['referenceLists/getMachinesTypesByZone'](zone);
    return machines.map(machine => ({
        label: `${machine.name} - ${machine.description}`,
        value: machine.name,
    }));
});
const diskTypes = computed(() => store.getters['referenceLists/getDiskTypes']);

// Fetch initial infrastructure status
const fetchInfrastructureStatus = async () => {
    try {
        const response = await api.get('/gcp/status');
        if (response.data?.gcp?.infrastructure_status) {
            infrastructureStatus.value = response.data.gcp.infrastructure_status;
        }
    } catch (error) {
        console.error('Failed to fetch infrastructure status:', error);
    }
};

// Connect to event WebSocket for real-time infrastructure status updates
const connectEventWebSocket = () => {
    const token = StorageService.get('token');
    const wsUrl = `${WS_BASE_URL}/events/stream?token=${token}`;

    eventWs.value = new WebSocket(wsUrl);

    eventWs.value.onopen = () => {
        console.log('Event WebSocket connected');
    };

    eventWs.value.onmessage = event => {
        try {
            const message = JSON.parse(event.data);

            if (message.type === 'infrastructure_status') {
                const newStatus = message.payload?.status;
                if (newStatus && message.payload?.provider === 'gcp') {
                    infrastructureStatus.value = newStatus;
                }
            }
        } catch (error) {
            console.error('Failed to parse event WebSocket message:', error);
        }
    };

    eventWs.value.onerror = error => {
        console.error('Event WebSocket error:', error);
    };

    eventWs.value.onclose = () => {
        console.log('Event WebSocket closed');
    };
};

onMounted(async () => {
    formFields.namePrefix.value = form.value.name_prefix;
    formFields.number.value = form.value.number;
    formFields.zone.value = form.value.zone;
    formFields.machineType.value = form.value.machine_type;
    formFields.role.value = form.value.role;
    formFields.diskSizeGb.value = form.value.disk_size_gb;
    formFields.diskType.value = form.value.disk_type;

    // Fetch initial status
    await fetchInfrastructureStatus();

    // Always connect to WebSocket to handle backend restarts
    connectEventWebSocket();
});

onUnmounted(() => {
    if (eventWs.value) {
        eventWs.value.close();
        eventWs.value = null;
    }
    if (ws.value) {
        ws.value.close();
        ws.value = null;
    }
});

// Form state
const formFields = reactive({
    namePrefix: new TextField({
        label: computed(() => t('provisioning.cloud.nodeFormfields.namePrefix')),
        type: 'text',
        required: true,
        rules: textfieldRules,
    }),
    number: new TextField({
        label: computed(() => t('provisioning.cloud.nodeFormfields.numberOfNodes')),
        type: 'number',
        min: 1,
        max: 10,
        required: true,
        rules: textfieldRules,
    }),
    role: new Select({
        label: computed(() => t('provisioning.cloud.nodeFormfields.role')),
        options: roleProvisioningTypes,
        required: true,
        rules: textfieldRules,
    }),
    zone: new AutoComplete({
        label: computed(() => t('provisioning.cloud.nodeFormfields.zone')),
        items: cloudZones,
        required: true,
        rules: textfieldRules,
    }),
    machineType: new AutoComplete({
        label: computed(() => t('provisioning.cloud.nodeFormfields.machineType')),
        items: availableMachineTypes,
        required: true,
        disabled: computed(() => !formFields.zone.value),
        rules: textfieldRules,
    }),
    diskSizeGb: new TextField({
        label: computed(() => t('provisioning.cloud.nodeFormfields.diskSizeGb')),
        type: 'number',
        min: 10,
        max: 65536,
        required: true,
        rules: textfieldRules,
    }),
    diskType: new Select({
        label: computed(() => t('provisioning.cloud.nodeFormfields.diskType')),
        options: diskTypes,
        required: true,
        rules: textfieldRules,
    }),
});

// Watch form fields and sync back to form
watch(
    () => formFields.namePrefix.value,
    newVal => {
        form.value.name_prefix = newVal;
    }
);
watch(
    () => formFields.number.value,
    newVal => {
        form.value.number = parseInt(newVal) || 1;
    }
);
watch(
    () => formFields.zone.value,
    newVal => {
        form.value.zone = newVal;
    }
);
watch(
    () => formFields.machineType.value,
    newVal => {
        form.value.machine_type = newVal;
    }
);
watch(
    () => formFields.role.value,
    newVal => {
        form.value.role = newVal;
    }
);
watch(
    () => formFields.diskSizeGb.value,
    newVal => {
        form.value.disk_size_gb = parseInt(newVal) || 100;
    }
);
watch(
    () => formFields.diskType.value,
    newVal => {
        form.value.disk_type = newVal;
    }
);

// Utility to scroll page to bottom
const scrollPageToBottom = () => {
    window.scrollTo({
        top: document.documentElement.scrollHeight,
        behavior: 'smooth',
    });
};

// Watch plan logs & scroll page as container grows
watch(
    () => planLogs.value.length,
    async () => {
        if (planLogs.value.length === 0) return;

        await nextTick();
        setTimeout(() => {
            scrollPageToBottom();
        }, 100);
    }
);

// Watch apply logs - scroll page as container grows
watch(
    () => applyLogs.value.length,
    async () => {
        if (applyLogs.value.length === 0) return;

        await nextTick();
        setTimeout(() => {
            scrollPageToBottom();
        }, 100);
    }
);

// Scroll page when phase changes
watch(provisioningPhase, async () => {
    await nextTick();
    setTimeout(() => {
        scrollPageToBottom();
    }, 300);
});

// Scroll page when resources are added
watch(
    () => planResources.value.length,
    async () => {
        if (planResources.value.length > 0) {
            await nextTick();
            setTimeout(() => {
                scrollPageToBottom();
            }, 300);
        }
    }
);

watch(
    () => applyResources.value.length,
    async () => {
        if (applyResources.value.length > 0) {
            await nextTick();
            setTimeout(() => {
                scrollPageToBottom();
            }, 300);
        }
    }
);

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

    // Reset resource tracking
    planResources.value = [];
    applyResources.value = [];
    resourcesMap.value = {};
    workflowData.value = null;

    try {
        // Step 1: POST to create provision request
        const response = await api.post('/gcp/nodes/provision', form.value);
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
    const token = StorageService.get('token');
    const wsUrl = `${WS_BASE_URL}/gcp/nodes/provision/${requestId}/stream?token=${token}`;

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
                const targetLogs =
                    provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete'
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
                const targetLogs =
                    provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete'
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
            } else if (message.type === 'resource_update') {
                // Handle resource updates
                const resource = message.payload;
                if (resource && resource.id) {
                    // Update or add resource in map
                    resourcesMap.value[resource.id] = resource;

                    // Separate resources by phase
                    const allResources = Object.values(resourcesMap.value);
                    if (
                        provisioningPhase.value === 'plan' ||
                        provisioningPhase.value === 'awaiting_approval'
                    ) {
                        planResources.value = allResources;
                    } else if (
                        provisioningPhase.value === 'apply' ||
                        provisioningPhase.value === 'complete'
                    ) {
                        applyResources.value = allResources;
                    }
                }
            } else if (message.type === 'workflow_update') {
                // Handle workflow updates
                workflowData.value = message.payload;

                // Update tracked resources if included in workflow
                if (message.payload && message.payload.resources) {
                    message.payload.resources.forEach(resource => {
                        if (resource && resource.id) {
                            resourcesMap.value[resource.id] = resource;
                        }
                    });

                    // Separate resources by phase
                    const allResources = Object.values(resourcesMap.value);
                    if (
                        provisioningPhase.value === 'plan' ||
                        provisioningPhase.value === 'awaiting_approval'
                    ) {
                        planResources.value = allResources;
                    } else if (
                        provisioningPhase.value === 'apply' ||
                        provisioningPhase.value === 'complete'
                    ) {
                        applyResources.value = allResources;
                    }
                }
            } else if (message.type === 'error') {
                // Route error to apply logs if we're in apply phase, otherwise to plan logs
                const targetLogs =
                    provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete'
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
        const targetLogs =
            provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete'
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
        const targetLogs =
            provisioningPhase.value === 'apply' || provisioningPhase.value === 'complete'
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
        const url = `/gcp/nodes/provision/${currentRequestId.value}/plan`;

        const response = await fetch(url, {
            headers: {
                Authorization: `Bearer ${token}`,
            },
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

const downloadApply = async () => {
    try {
        const token = StorageService.get('token');
        const url = `/gcp/nodes/provision/${currentRequestId.value}/apply`;

        const response = await fetch(url, {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        });

        if (!response.ok) {
            throw new Error('Failed to download apply logs');
        }

        const blob = await response.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = downloadUrl;
        link.download = `apply-${currentRequestId.value}.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(downloadUrl);
    } catch (error) {
        console.error('Failed to download apply logs:', error);
    }
};
</script>
