<template>
    <v-card v-if="hasResources || workflow" class="resources-compact elevation-2">
        <!-- Header with title and summary counts -->
        <v-card-title class="d-flex align-center justify-space-between pa-3">
            <div class="d-flex align-center">
                <v-icon class="mr-2" size="20">{{
                    phase === 'apply' ? 'mdi-cloud-check' : 'mdi-cloud-sync'
                }}</v-icon>
                <span class="text-h6">{{ title || $t('provisioning.cloud.resources.title') }}</span>
            </div>

            <!-- Summary counts inline -->
            <div v-if="workflow?.summary" class="d-flex align-center gap-3">
                <div
                    v-for="(count, action) in workflow.summary"
                    :key="action"
                    class="d-flex align-center"
                >
                    <v-icon :color="getActionColor(action)" size="18" class="mr-1">
                        {{ getActionIcon(action) }}
                    </v-icon>
                    <span class="text-body-2 font-weight-medium">{{ count }}</span>
                </div>
            </div>
        </v-card-title>

        <v-divider />

        <!-- Compact resource list -->
        <v-card-text class="pa-0">
            <!-- Resource list -->
            <v-list density="compact" class="py-0">
                <template v-for="(resources, type) in groupedResources" :key="type">
                    <!-- Type header -->
                    <v-list-subheader class="text-caption text-uppercase px-3 py-1">
                        <v-icon size="14" class="mr-1">{{ getTypeIcon(type) }}</v-icon>
                        {{ type }}
                    </v-list-subheader>

                    <!-- Resources with inline expandable details -->
                    <template v-for="resource in resources" :key="resource.id">
                        <v-list-item
                            class="px-3 py-1 resource-item"
                            @click="toggleResourceExpansion(resource.id)"
                        >
                            <!-- Resource name and status -->
                            <template v-slot:prepend>
                                <v-icon
                                    :color="
                                        phase === 'plan'
                                            ? getActionColor(resource.action)
                                            : getStatusColor(resource.status)
                                    "
                                    size="16"
                                >
                                    {{ getResourceIcon(resource.type) }}
                                </v-icon>
                            </template>

                            <v-list-item-title class="text-body-2">
                                {{ resource.name }} {{ resource.details?.name || '' }}
                            </v-list-item-title>

                            <v-list-item-subtitle class="text-caption">
                                {{ $t(`provisioning.cloud.resources.actions.${resource.action}`) }}
                            </v-list-item-subtitle>

                            <template v-slot:append>
                                <div class="d-flex align-center">
                                    <!-- Show action in plan phase, status in apply phase -->
                                    <v-chip
                                        v-if="phase === 'plan'"
                                        :color="getActionColor(resource.action)"
                                        size="x-small"
                                        variant="tonal"
                                        class="font-weight-medium mr-2"
                                    >
                                        {{
                                            $t(
                                                `provisioning.cloud.resources.actions.${resource.action}`
                                            )
                                        }}
                                    </v-chip>
                                    <v-chip
                                        v-else
                                        :color="getStatusColor(resource.status)"
                                        size="x-small"
                                        variant="tonal"
                                        class="font-weight-medium mr-2"
                                    >
                                        {{
                                            $t(
                                                `provisioning.cloud.resources.statuses.${resource.status}`
                                            )
                                        }}
                                    </v-chip>
                                    <v-icon size="16">
                                        {{
                                            expandedResources[resource.id]
                                                ? 'mdi-chevron-up'
                                                : 'mdi-chevron-down'
                                        }}
                                    </v-icon>
                                </div>
                            </template>
                        </v-list-item>

                        <!-- Inline expandable details for each resource -->
                        <v-expand-transition>
                            <v-sheet
                                v-if="expandedResources[resource.id]"
                                color="grey-darken-4"
                                class="px-4 py-2 resource-inline-details"
                            >
                                <v-row dense class="mt-1">
                                    <v-col cols="12">
                                        <div class="text-caption text-grey">
                                            {{
                                                $t(
                                                    'provisioning.cloud.resources.details.resourceId'
                                                )
                                            }}
                                        </div>
                                        <div class="text-caption font-mono">{{ resource.id }}</div>
                                    </v-col>
                                </v-row>

                                <v-row dense v-if="resource.started_at || resource.duration">
                                    <v-col cols="6">
                                        <div class="text-caption text-grey">
                                            {{ $t('provisioning.cloud.resources.details.started') }}
                                        </div>
                                        <div class="text-caption">
                                            {{ formatTime(resource.started_at) }}
                                        </div>
                                    </v-col>
                                    <v-col cols="6">
                                        <div class="text-caption text-grey">
                                            {{
                                                $t('provisioning.cloud.resources.details.duration')
                                            }}
                                        </div>
                                        <div class="text-caption">
                                            {{
                                                resource.duration ||
                                                $t(
                                                    'provisioning.cloud.resources.details.inProgress'
                                                )
                                            }}
                                        </div>
                                    </v-col>
                                </v-row>

                                <div v-if="resource.error" class="mt-2">
                                    <v-alert
                                        type="error"
                                        density="compact"
                                        variant="tonal"
                                        class="text-caption"
                                    >
                                        {{ resource.error }}
                                    </v-alert>
                                </div>

                                <div
                                    v-if="
                                        resource.details && Object.keys(resource.details).length > 0
                                    "
                                    class="mt-2"
                                >
                                    <div class="text-caption text-grey mb-1">
                                        {{
                                            $t('provisioning.cloud.resources.details.configuration')
                                        }}
                                    </div>
                                    <pre class="text-caption overflow-auto details-pre">{{
                                        JSON.stringify(resource.details, null, 2)
                                    }}</pre>
                                </div>
                            </v-sheet>
                        </v-expand-transition>
                    </template>
                </template>

                <!-- Empty state -->
                <v-list-item v-if="!hasResources" class="text-center py-4">
                    <v-list-item-title class="text-caption text-grey">
                        {{
                            emptyMessage ||
                            $t('provisioning.cloud.resources.emptyMessages.duringProvisioning')
                        }}
                    </v-list-item-title>
                </v-list-item>
            </v-list>

            <!-- Outputs Section -->
            <div
                v-if="workflow?.outputs && Object.keys(workflow.outputs).length > 0"
                class="pa-3 mt-2 bg-grey-darken-4"
            >
                <div class="text-caption text-grey mb-2">
                    <v-icon size="14" class="mr-1">mdi-export-variant</v-icon>
                    Outputs
                </div>
                <div v-for="(value, key) in workflow.outputs" :key="key" class="mb-1">
                    <span class="text-caption font-weight-medium">{{ key }}:</span>
                    <span class="text-caption ml-2 font-mono">{{ formatOutputValue(value) }}</span>
                </div>
            </div>
        </v-card-text>
    </v-card>
</template>

<script setup>
import { ref, computed } from 'vue';

const props = defineProps({
    resources: {
        type: Array,
        default: () => [],
    },
    workflow: {
        type: Object,
        default: null,
    },
    title: {
        type: String,
        default: null,
    },
    phase: {
        type: String,
        default: 'plan',
    },
    emptyMessage: {
        type: String,
        default: null,
    },
});

const expandedResources = ref({});

// Check if we have resources to display
const hasResources = computed(() => props.resources && props.resources.length > 0);

// Group resources by type
const groupedResources = computed(() => {
    if (!hasResources.value) return {};

    const groups = {};
    props.resources.forEach(resource => {
        const type = resource.type || 'other';
        if (!groups[type]) {
            groups[type] = [];
        }
        groups[type].push(resource);
    });

    // Sort resources within each group
    Object.keys(groups).forEach(type => {
        groups[type].sort((a, b) => a.name.localeCompare(b.name));
    });

    return groups;
});

// Get colors based on action/status
const getActionColor = action => {
    switch (action) {
        case 'create':
            return 'success';
        case 'update':
            return 'blue';
        case 'delete':
            return 'error';
        default:
            return 'grey';
    }
};

const getStatusColor = status => {
    switch (status) {
        case 'creating':
        case 'modifying':
            return 'blue';
        case 'complete':
            return 'success';
        case 'failed':
            return 'error';
        case 'deleting':
            return 'orange';
        default:
            return 'grey';
    }
};

const toggleResourceExpansion = resourceId => {
    expandedResources.value[resourceId] = !expandedResources.value[resourceId];
};

const formatTime = timestamp => {
    if (!timestamp) return '';
    return new Date(timestamp).toLocaleTimeString();
};

// Get icons
const getActionIcon = action => {
    switch (action) {
        case 'create':
            return 'mdi-plus-circle-outline';
        case 'update':
            return 'mdi-pencil-circle-outline';
        case 'delete':
            return 'mdi-delete-circle-outline';
        default:
            return 'mdi-help-circle-outline';
    }
};

const getTypeIcon = type => {
    // For now we only differentiate vm instances
    if (type && type.includes('instance')) {
        return 'mdi-server';
    }
    return 'mdi-cube-outline';
};

const getResourceIcon = type => {
    return getTypeIcon(type);
};

const formatOutputValue = value => {
    if (typeof value === 'object' && value !== null) {
        if (value.value !== undefined) {
            return formatOutputValue(value.value);
        }
        return JSON.stringify(value, null, 2);
    }
    return String(value);
};
</script>

<style scoped>
.resources-compact {
    max-width: 100%;
    overflow: hidden;
}

.resource-item {
    transition: background-color 0.2s ease;
    cursor: pointer;
    min-height: 40px !important;
}

.resource-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.resource-inline-details {
    border-left: 2px solid rgba(255, 255, 255, 0.2);
    margin-left: 12px;
}

.font-mono {
    font-family: monospace;
    font-size: 11px;
}

.details-pre {
    margin: 0;
    white-space: pre-wrap;
    word-wrap: break-word;
    max-height: 150px;
    font-size: 11px;
}

.gap-3 {
    gap: 12px;
}

.v-list-item {
    min-height: 36px !important;
}

.v-list-subheader {
    min-height: 24px !important;
    background-color: rgba(255, 255, 255, 0.02);
}
</style>
