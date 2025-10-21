<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('provisioning.cloud.title')"
            :subheading="$t('provisioning.cloud.subheading')"
        />

        <!-- Simple test form for WebSocket provisioning -->
        <v-container>
            <v-card class="pa-6 mb-4">
                <v-card-title>GCP Node Provisioning Test</v-card-title>
                <v-card-text>
                    <v-form @submit.prevent="submitProvisionRequest">
                        <v-text-field
                            v-model="form.name_prefix"
                            label="Name Prefix"
                            hint="e.g., worker"
                            required
                        ></v-text-field>

                        <v-text-field
                            v-model.number="form.number"
                            label="Number of Nodes"
                            type="number"
                            min="1"
                            max="20"
                            required
                        ></v-text-field>

                        <v-text-field
                            v-model="form.zone"
                            label="Zone"
                            hint="e.g., us-central1-a"
                            required
                        ></v-text-field>

                        <v-text-field
                            v-model="form.machine_type"
                            label="Machine Type"
                            hint="e.g., n1-standard-2"
                            required
                        ></v-text-field>

                        <v-select
                            v-model="form.role"
                            :items="['worker', 'control-plane']"
                            label="Role"
                            required
                        ></v-select>

                        <v-btn type="submit" color="primary" :disabled="isProvisioning" block>
                            {{ isProvisioning ? 'Provisioning...' : 'Provision Nodes' }}
                        </v-btn>
                    </v-form>
                </v-card-text>
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
        </v-container>

        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular indeterminate></v-progress-circular>
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import { ref } from 'vue';
import api from '@/services/api';
import { StorageService } from '@/services/storage.service';

const overlay = ref(false);
const isProvisioning = ref(false);
const planLogs = ref([]);
const applyLogs = ref([]);
const status = ref('');
const ws = ref(null);
const provisioningPhase = ref('idle'); // idle, plan, awaiting_approval, apply, complete, error
const approvalSummary = ref('');

const form = ref({
    name_prefix: 'worker',
    number: 2,
    zone: 'us-central1-a',
    machine_type: 'n1-standard-2',
    role: 'worker',
});

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

            // Determine which log array to use based on the current phase
            const currentLogs = provisioningPhase.value === 'apply' ? applyLogs : planLogs;

            if (message.type === 'log') {
                currentLogs.value.push({
                    type: 'log',
                    message: message.payload.message,
                    timestamp: new Date(),
                });
            } else if (message.type === 'status') {
                status.value = message.payload.status;

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

                currentLogs.value.push({
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
                currentLogs.value.push({
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
        const currentLogs = provisioningPhase.value === 'apply' ? applyLogs : planLogs;
        currentLogs.value.push({
            type: 'error',
            message: `WebSocket error: ${error}`,
            timestamp: new Date(),
        });
        provisioningPhase.value = 'error';
        isProvisioning.value = false;
    };

    ws.value.onclose = () => {
        const currentLogs = provisioningPhase.value === 'apply' ? applyLogs : planLogs;
        currentLogs.value.push({
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
</script>
