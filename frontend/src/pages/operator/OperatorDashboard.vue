<template>
    <div>
        <BaseDataTable
            v-model="search"
            :headers="nodeHeaders"
            :items="nodes"
            :loading="loading"
            :loadingText="$t('dashboard.provision.table.loadingText')"
            :noDataText="$t('dashboard.provision.table.noDataText')"
            :itemsPerPageText="$t('dashboard.provision.table.itemsPerPageText')"
            :titleToolbar="$t('dashboard.provision.table.title')"
            :actionsButtonForTable="actionsButtonForTable"
            rowClickable
            @click:row="(event, item) => showDetailsNodeDialog(item.item)"
        >
            <template #[`item.status`]="{ item }">
                <v-chip :color="getStatusColor(item.status)">
                    {{ item.status }}
                </v-chip>
            </template>        
            <template #[`item.labels`]="{ item }">
                <v-chip 
                    v-for="(label, index) in item.labels"
                    :key="index"
                    class="ma-1"
                >
                    {{ label }}
                </v-chip>
            </template>
        </BaseDataTable>
        <ViewDetailsNodeDialog v-model="dialogViewDetailsNode" :node="selectedNode" />
    </div>
</template>

<script setup>
import { getConnectedNodes } from '@/services/provisioning.service';
import { computed, ref , onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import ViewDetailsNodeDialog from './dialogs/node/ViewDetailsNodeDialog.vue';

const { t } = useI18n();

// State
const search = ref('');
const loading = ref(false);
const nodes = ref([]);
const dialogViewDetailsNode = ref(false);
const selectedNode = ref(null);

// Computed
const nodeHeaders = computed(() => [
    { title: t('dashboard.provision.table.headers.nodename'), value: 'name' },
    { title: t('dashboard.provision.table.headers.role'), value: 'role', align : "center" },
    { title: t('dashboard.provision.table.headers.provider'), value: 'provider', align : "center" },
    { title: t('dashboard.provision.table.headers.status'), value: 'status', align: "center" },
    { title: t('dashboard.provision.table.headers.labels'), value: 'labels', align: "center" },
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchConnectedNodes
    }
]);

//mounted
onMounted(() => {
    fetchConnectedNodes();
});

// Methods
function fetchConnectedNodes() {
    loading.value = true;

    getConnectedNodes()
        .then(response => {
            nodes.value = response
                .filter(node => node.status?.toLowerCase() !== "pending")
                .map(node => ({
                    ...node,
                    status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                    role: node.role.charAt(0).toUpperCase() + node.role.slice(1),
                    provider: node.provider.charAt(0).toUpperCase() + node.provider.slice(1),
                    labels: JSON.parse(node.labels || '[]'),
                }));
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
function getStatusColor(status) {
    switch (status.toLowerCase()) {
        case 'active':
            return 'success';
        case 'provisioning':
            return 'warning';
        case 'failed':
            return 'error';
        default:
            return 'grey';
    }
}
</script>