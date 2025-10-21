<template>
    <v-sheet class="mt-4 border rounded">
        <v-data-table
        :headers="nodeHeaders"
        :items="nodes"
        :items-length="nodes.length"
        :search="search"
        :loading="loading"
        :loading-text="$t('dashboard.provision.table.loadingText')"
        :no-data-text="$t('dashboard.provision.table.noDataText')"
        :items-per-page="10"
        :items-per-page-text="$t('dashboard.provision.table.itemsPerPageText')"
        :hide-default-footer="nodes.length < 10"
        mobile-breakpoint="md"
        >
            <!-- Slot for top -->
            <template v-slot:top>
                <BaseToolbarTable :title="$t('dashboard.provision.table.title')" :buttons="actionsButtonForTable" />
                <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined" hide-details single-line dense class="pa-3"/>
            </template>

            <!-- Slot for status -->
            <template #[`item.status`]="{ item }">
                <v-chip :color="getStatusColor(item.status)">
                    {{ item.status }}
                </v-chip>
            </template>

            <!-- Slot for labels -->
            <template #[`item.labels`]="{ item }">
                <v-chip 
                    v-for="(label, index) in item.labels"
                    :key="index"
                    class="ma-1"
                >
                    {{ label }}
                </v-chip>
            </template>
        </v-data-table>
    </v-sheet>
</template>

<script setup>
import { getConnectedNodes } from '@/services/provisioning.service';
import { computed, ref , onMounted } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

// State
const search = ref('');
const loading = ref(false);
const nodes = ref([]);

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