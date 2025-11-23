<template>
    <div>
        <NodesCharts :nodes="nodesChart"></NodesCharts>
        <BaseDataTable v-model="search" :headers="nodeHeaders" :items="nodes" :loading="loading"
            :loadingText="$t('dashboard.provision.table.loadingText')"
            :noDataText="$t('dashboard.provision.table.noDataText')"
            :itemsPerPageText="$t('dashboard.provision.table.itemsPerPageText')"
            :titleToolbar="$t('dashboard.provision.table.title')" :actionsButtonForTable="actionsButtonForTable"
            rowClickable @click:row="(event, item) => showDetailsNodeDialog(item.item)">
            <template #[`item.status`]="{ item }">
                <BaseChip :color="getStatusColor(item.status)">
                    <template #prepend>
                        <v-progress-circular v-if="normalizeStatus(item.status) === 'provisioning'" indeterminate size="14" width="2"/>
                        <v-icon v-else-if="normalizeStatus(item.status) === 'active'" size="14">mdi-check-circle-outline</v-icon>
                        <v-icon v-else-if="normalizeStatus(item.status) === 'failed'" size="14">mdi-close-circle-outline</v-icon>
                    </template>
                    <span>{{ item.status }}</span>
                </BaseChip>
            </template>
            <template #[`item.labels`]="{ item }">
                <BaseChip v-for="(label, index) in item.labels" :key="index" :color="getLabelColor(label)">
                    {{ label }}
                </BaseChip>
            </template>
        </BaseDataTable>
        <ViewDetailsNodeDialog v-model="dialogViewDetailsNode" :node="selectedNode" />
    </div>
</template>

<script setup>
import { getConnectedNodes } from '@/services/provisioning.service';
import { computed, ref, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import { StatusColorHandler } from '@/composables/StatusColorHandler';
import { LabelColorHandler } from '@/composables/LabelColorHandler';
import ViewDetailsNodeDialog from '@/pages/dialogs/node/ViewDetailsNodeDialog.vue';
import wsEventService from '@/services/wsEvent.service';

const { t } = useI18n();
const { getStatusColor } = StatusColorHandler();
const { getLabelColor } = LabelColorHandler();

// State
const search = ref('');
const loading = ref(false);
const nodes = ref([]);
const nodesChart = ref([]);
const dialogViewDetailsNode = ref(false);
const selectedNode = ref(null);

// Computed
const nodeHeaders = computed(() => [
    { title: t('dashboard.provision.table.headers.nodename'), value: 'name' },
    { title: t('dashboard.provision.table.headers.role'), value: 'role', align: "center" },
    { title: t('dashboard.provision.table.headers.provider'), value: 'provider', align: "center" },
    { title: t('dashboard.provision.table.headers.status'), value: 'status', align: "center" },
    { title: t('dashboard.provision.table.headers.labels'), value: 'labels', width: "25%" },
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchConnectedNodes
    }
]);

let unsubscribeNodeStatusUpdated;

//mounted
onMounted(() => {
    fetchConnectedNodes();
    unsubscribeNodeStatusUpdated = wsEventService.subscribe('NodeStatusUpdated', () => {
        fetchConnectedNodes();
    });
});

onBeforeUnmount(() => {
    if (typeof unsubscribeNodeStatusUpdated === 'function') {
        unsubscribeNodeStatusUpdated();
    }
});

// Methods
function normalizeStatus(status) {
    return typeof status === 'string' ? status.toLowerCase() : '';
}

function fetchConnectedNodes() {
    loading.value = true;

    getConnectedNodes()
        .then(response => {
            nodesChart.value = response
                .map(node => ({
                    ...node,
                    status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                    role: node.role.charAt(0).toUpperCase() + node.role.slice(1),
                    provider: node.provider.charAt(0).toUpperCase() + node.provider.slice(1),
                    labels: JSON.parse(node.labels || '[]'),
                }));

            nodes.value = nodesChart.value.filter(node => node.status?.toLowerCase() !== "pending");
        })
        .catch(error => {
            console.error('Error fetching connected nodes active:', error);
        })
        .finally(() => {
            loading.value = false;
        });
}
function showDetailsNodeDialog(node) {
    selectedNode.value = node;
    dialogViewDetailsNode.value = true;
}
</script>
