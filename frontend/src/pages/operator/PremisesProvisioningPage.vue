<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.onPremises.title')"
            :subheading="$t('provisioning.onPremises.subheading')"
            :actions="actionsLabelBar"
        />
        <BaseDataTable
            :headers="nodeHeaders"
            :items="nodes"
            :loading="loading"
            :loadingText="$t('provisioning.onPremises.table.loadingText')"
            :noDataText="$t('provisioning.onPremises.table.noDataText')"
            :itemsPerPageText="$t('provisioning.onPremises.table.itemsPerPageText')"
            :titleToolbar="$t('provisioning.onPremises.table.title')"
            :actionsButtonForTable="actionsButtonForTable"
        >
            <!-- Slot for status -->
            <template #[`item.status`]="{ item }">
                <v-chip :color="getStatusColor(item.status)">
                    {{ item.status }}
                </v-chip>
            </template>
            
            <!-- Slot for roles -->
            <template #[`item.role`]="{ item }">
                <v-select
                v-model="item.role"
                :items="provisioningRoles"
                item-value="value"
                item-title="label"
                dense
                density="compact"
                placeholder="Select role"
                variant="solo"
                hide-details
                ></v-select>
            </template>

            <!-- Slot for labels -->
            <template #[`item.labels`]="{ item }">
                <div class="d-flex flex-wrap align-center">
                    <v-chip
                        v-for="(label, index) in item.labels"
                        :key="index"
                        class="ma-1"
                        closable
                        @click:close="item.labels.splice(index, 1)"
                    >
                        {{ label }}
                    </v-chip>
                    <template v-if="!item.addingLabel">
                        <v-chip class="ma-1" elevation="2" @click="item.addingLabel = true">
                            {{ $t('provisioning.onPremises.buttons.addLabel') }}
                        </v-chip>
                    </template>
                    <template v-else>
                        <v-text-field
                            v-model="item.newLabel"
                            density="compact"
                            placeholder="New label"
                            variant="solo"
                            rounded
                            hide-details
                            max-width="120"
                            autofocus
                            @keyup.enter="addLabel(item)"
                            @blur="addLabel(item); item.addingLabel = false"
                        />
                    </template>
                </div>
            </template>
        </BaseDataTable>
        <DownloadISOOnPremDialog v-model="dialogDownloadISOOnPremise" />
    </PortalLayout>
</template>

<script setup>
import { getConnectedNodes, provisionNodes } from '@/services/provisioning.service';
import { onMounted, ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import { StatusColorHandler } from '@/composables/StatusColorHandler';
import DownloadISOOnPremDialog from '@/pages/operator/dialogs/download/DownloadISOOnPremDialog.vue';

const { t } = useI18n();
const store = useStore();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();
const { getStatusColor } = StatusColorHandler();

// State
const loading = ref(false);
const nodes = ref([]);
const dialogDownloadISOOnPremise = ref(false);

// Mounted
onMounted(() => {
    fetchConnectedNodes();
});

// Computed
const actionsLabelBar = computed(() => [
    { icon: "mdi-download", text: t('actionButtons.downloadISOOnPremise'), tooltip: t('actionButtons.downloadISOOnPremise'), onClick: () => showDownloadISODialog() }
]);
const nodeHeaders = computed(() => [
    { title: t('provisioning.onPremises.table.headers.ip'), value: 'ip_address'},
    { title: t('provisioning.onPremises.table.headers.mac'), value: 'mac_address', width: "20%" },
    { title: t('provisioning.onPremises.table.headers.status'), value: 'status', width: "15%" },
    { title: t('provisioning.onPremises.table.headers.role'), value: 'role', width: "20%" },
    { title: t('provisioning.onPremises.table.headers.labels'), value: 'labels', width: "30%" },
]);
const provisioningRoles = computed(() => store.getters['referenceLists/getProvisioningRoles']);
const canProvision = computed(() => {
    if (!nodes.value.length) return false

    return nodes.value.every(node =>
        node.role && node.labels && node.labels.length > 0
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
function fetchConnectedNodes() {
    loading.value = true;
    getConnectedNodes({status: "pending"})
        .then(response => {        
            nodes.value = response
                .filter(node => node.provider?.toLowerCase() === "onprem")
                .map(node => ({
                    ...node,
                    status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                    role: null,
                    labels: [],
                }));
        })
        .catch(error => {
            console.error('Error fetching connected nodes:', error);
        })
        .finally(() => {
            loading.value = false;
        });
}
function addLabel(item) {
    if (item.newLabel && !item.labels.includes(item.newLabel)) {
        item.labels.push(item.newLabel);
    }
    item.newLabel = '';
}
function provisionConnectedNodes() {
    if (!canProvision.value) return;
    showOverlay();

    const payloadNodes = nodes.value.map(node => ({
        node_id: node.id,
        role: node.role,
        labels: node.labels,
    }));

    provisionNodes({ nodes: payloadNodes })
    .then(() => {
        showNotification(t('provisioning.onPremises.notifications.provisionSuccess'), 'success');
    })
    .catch(error => {
        console.error('Error provisioning connected nodes:', error);
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