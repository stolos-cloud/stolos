<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('provisioning.onPremises.title')"
            :subheading="$t('provisioning.onPremises.subheading')" :actions="actionsLabelBar" />
        <BaseDataTable :headers="nodeHeaders" :items="nodes" :loading="loading"
            :loadingText="$t('provisioning.onPremises.table.loadingText')"
            :noDataText="$t('provisioning.onPremises.table.noDataText')"
            :itemsPerPageText="$t('provisioning.onPremises.table.itemsPerPageText')"
            :titleToolbar="$t('provisioning.onPremises.table.title')" :actionsButtonForTable="actionsButtonForTable">
            <!-- Slot for status -->
            <template #[`item.status`]="{ item }">
                <v-chip :color="getStatusColor(item.status)">
                    {{ item.status }}
                </v-chip>
            </template>

            <template #[`item.installDisk`]="{ item }">
                <v-select v-model="item.installDisk" :items="item.diskOptions" :loading="item.disksLoading"
                    density="compact" :placeholder="$t('provisioning.onPremises.table.headers.disk')" variant="solo"
                    hide-details :disabled="item.disksLoading || !item.diskOptions.length" />
            </template>

            <!-- Slot for roles -->
            <template #[`item.role`]="{ item }">
                <v-select v-model="item.role" :items="provisioningRoles" item-value="value" item-title="label" dense
                    density="compact" placeholder="Select role" variant="solo" hide-details></v-select>
            </template>

            <!-- Slot for labels -->
            <template #[`item.labels`]="{ item }">
                <div class="d-flex flex-wrap align-center">
                    <v-chip v-for="(label, index) in item.labels" :key="index" :color="getLabelColor(label)" class="ma-1" label closable
                        @click:close="item.labels.splice(index, 1)">
                        {{ label }}
                    </v-chip>
                    <template v-if="!item.addingLabel">
                        <v-chip class="ma-1" elevation="2" @click="item.addingLabel = true">
                            {{ $t('provisioning.onPremises.buttons.addLabel') }}
                        </v-chip>
                    </template>
                    <template v-else>
                        <v-text-field v-model="item.newLabel" density="compact" placeholder="New label" variant="solo"
                            rounded hide-details max-width="120" autofocus @keyup.enter="addLabel(item)"
                            @blur="addLabel(item); item.addingLabel = false" />
                    </template>
                </div>
            </template>
            <template #bottom>
                <v-divider class="mb-4"></v-divider>
                <div style="padding-bottom: 10px;" class="d-flex justify-center align-center text-center">
                    <v-progress-circular indeterminate class="mr-2" size="20"></v-progress-circular>
                    <span class="text-body-2">{{ $t('provisioning.onPremises.table.footerSpinnerMsg') }}</span>
                </div>
            </template>
        </BaseDataTable>
        <DownloadISOOnPremDialog v-model="dialogDownloadISOOnPremise" />
    </PortalLayout>
</template>

<script setup>
import { getConnectedNodes, getNodeDisks, provisionNodes } from '@/services/provisioning.service';
import { onMounted, onBeforeUnmount, ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import { StatusColorHandler } from '@/composables/StatusColorHandler';
import { LabelColorHandler } from '@/composables/LabelColorHandler';
import DownloadISOOnPremDialog from '@/pages/operator/dialogs/download/DownloadISOOnPremDialog.vue';
import wsEventService from '@/services/wsEvent.service';

const { t } = useI18n();
const store = useStore();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();
const { getStatusColor } = StatusColorHandler();
const { getLabelColor } = LabelColorHandler();

// State
const loading = ref(false);
const nodes = ref([]);
const dialogDownloadISOOnPremise = ref(false);

let unsubscribeNodeStatusUpdated;
let unsubscribeNewPendingNodeDetected;

// Mounted
onMounted(() => {
    fetchConnectedNodes();
    unsubscribeNodeStatusUpdated = wsEventService.subscribe('NodeStatusUpdated', () => {
        fetchConnectedNodes();
    });
    unsubscribeNewPendingNodeDetected = wsEventService.subscribe('NewPendingNodeDetected', () => {
        fetchConnectedNodes();
    });
});

onBeforeUnmount(() => {
    if (typeof unsubscribeNodeStatusUpdated === 'function') {
        unsubscribeNodeStatusUpdated();
    }
    if (typeof unsubscribeNewPendingNodeDetected === 'function') {
        unsubscribeNewPendingNodeDetected();
    }
});

// Computed
const actionsLabelBar = computed(() => [
    { icon: "mdi-download", text: t('actionButtons.downloadISOOnPremise'), tooltip: t('actionButtons.downloadISOOnPremise'), onClick: () => showDownloadISODialog() }
]);
const nodeHeaders = computed(() => [
    { title: t('provisioning.onPremises.table.headers.ip'), value: 'ip_address' },
    { title: t('provisioning.onPremises.table.headers.mac'), value: 'mac_address', width: "20%" },
    { title: t('provisioning.onPremises.table.headers.status'), value: 'status', width: "15%" },
    { title: t('provisioning.onPremises.table.headers.disk'), value: 'installDisk', width: "20%" },
    { title: t('provisioning.onPremises.table.headers.role'), value: 'role', width: "20%" },
    { title: t('provisioning.onPremises.table.headers.labels'), value: 'labels', width: "30%" },
]);
const provisioningRoles = computed(() => store.getters['referenceLists/getProvisioningRoles']);
const canProvision = computed(() => {
    if (!nodes.value.length) return false

    return nodes.value.every(node =>
        node.role && node.installDisk
    )
});
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchConnectedNodes
    },
    {
        icon: "mdi-server-plus",
        tooltip: t('provisioning.onPremises.buttons.provisionConnectedNodes'),
        text: t('provisioning.onPremises.buttons.provisionConnectedNodes'),
        disabled: !canProvision.value,
        click: provisionConnectedNodes
    }
]);

// Methods
function showDownloadISODialog() {
    dialogDownloadISOOnPremise.value = true;
}
async function fetchConnectedNodes() {
    loading.value = true;
    try {
        const response = await getConnectedNodes({ status: "pending" });
        nodes.value = response
            .filter(node => node.provider?.toLowerCase() === "onprem")
            .map(node => ({
                ...node,
                status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                role: null,
                labels: [],
                installDisk: null,
                diskOptions: [],
                disksLoading: false,
            }));
        await Promise.all(nodes.value.map(node => loadNodeDisks(node)));
    } catch (error) {
        console.error('Error fetching connected nodes:', error);
    } finally {
        loading.value = false;
    }
}
function addLabel(item) {
    if (item.newLabel && !item.labels.includes(item.newLabel)) {
        item.labels.push(item.newLabel);
    }
    item.newLabel = '';
}
async function loadNodeDisks(node) {
    node.disksLoading = true;
    try {
        const disks = await getNodeDisks(node.id);
        node.diskOptions = disks;
        if (!node.installDisk && disks.length) {
            node.installDisk = disks[0];
        }
    } catch (error) {
        console.error(`Error fetching disks for node ${node.id}:`, error);
        node.diskOptions = [];
        node.installDisk = null;
    } finally {
        node.disksLoading = false;
    }
}
function provisionConnectedNodes() {
    if (!canProvision.value) return;
    showOverlay();

    const payloadNodes = nodes.value.map(node => ({
        node_id: node.id,
        role: node.role,
        labels: node.labels,
        install_disk: node.installDisk,
    }));

    provisionNodes({ nodes: payloadNodes })
        .then((results) => {
            // Check individual node results
            const failures = results.filter(r => !r.succeeded);

            if (failures.length === 0) {
                showNotification(t('provisioning.onPremises.notifications.provisionSuccess'), 'success');
                // Refresh the node list after successful provisioning
                fetchConnectedNodes();
            } else if (failures.length === results.length) {
                // All failed
                const errorMsg = failures.map(f => `${f.node_id}: ${f.error}`).join('; ');
                showNotification(t('provisioning.onPremises.notifications.provisionFailed', { error: errorMsg }), 'error');
            } else {
                // Partial success
                showNotification(
                    t('provisioning.onPremises.notifications.provisionPartial', {
                        success: results.length - failures.length,
                        total: results.length,
                        failures: failures.length
                    }),
                    'warning'
                );
                fetchConnectedNodes();
            }
        })
        .catch(error => {
            console.error('Error provisioning connected nodes:', error);
            showNotification(`Provisioning error: ${error.message || error}`, 'error');
        })
        .finally(() => {
            hideOverlay();
        });
}
</script>

<style scoped>
.chip-input .v-field__input {
    padding: 0 8px !important;
}
</style>
