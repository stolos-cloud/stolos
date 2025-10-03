<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.onPremises.title')"
            :subheading="$t('provisioning.onPremises.subheading')"
        />
        <div class="mt-4">
            <h3>{{ $t('provisioning.onPremises.table.title') }}</h3>
            <v-data-table-server
                :headers="nodeHeaders"
                :items="nodes"
                :items-length="nodes.length"
                :loading=loading
                :loading-text="$t('provisioning.onPremises.table.loadingText')"
                :no-data-text="$t('provisioning.onPremises.table.noDataText')"
                :items-per-page="10"
                :items-per-page-text="$t('provisioning.onPremises.table.itemsPerPageText')"
                class="elevation-8 mt-2"
                mobile-breakpoint="md"
                disable-sort="true"
            >
                <!-- Slot for status -->
                <template #item.status="{ item }">
                    <v-chip color="primary">
                        {{ item.status }}
                    </v-chip>
                </template>
                
                <!-- Slot for roles -->
                <template #item.role="{ item }">
                    <v-select
                    v-model="item.role"
                    :items="roles"
                    item-value="key"
                    item-title="title"
                    dense
                    density="compact"
                    placeholder="Select role"
                    variant="solo"
                    hide-details
                    ></v-select>
                </template>

                <!-- Slot for labels -->
                <template #item.labels="{ item }">
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
            </v-data-table-server>

            <div class="d-flex justify-end">
                <BaseButton :text="$t('provisioning.onPremises.buttons.provisionConnectedNodes')" color="primary" class="mt-2" :disabled="!canProvision" @click="provisionConnectedNodes" />
            </div>
        </div>
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular
                indeterminate
            ></v-progress-circular>
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import { getConnectedNodes, createNodesWithRoleAndLabels } from '@/services/provisioning.service';
import { onMounted, ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

const { t } = useI18n();
const route = useRouter();

const loading = ref(false);
const overlay = ref(false);
const nodes = ref([]);
const roles = [
    { key: 'worker', title: 'Worker' },
    { key: 'control-plane', title: 'Control plane' },
];

//mounted
onMounted(() => {
    fetchConnectedNodes();
});

// Computed
const nodeHeaders = computed(() => [
  { title: t('provisioning.onPremises.table.headers.ip'), value: 'ip_address'},
  { title: t('provisioning.onPremises.table.headers.mac'), value: 'mac_address', width: "20%" },
  { title: t('provisioning.onPremises.table.headers.status'), value: 'status', width: "15%" },
  { title: t('provisioning.onPremises.table.headers.role'), value: 'role', width: "20%" },
  { title: t('provisioning.onPremises.table.headers.labels'), value: 'labels', width: "30%" },
]);
const canProvision = computed(() => {
  if (!nodes.value.length) return false

  return nodes.value.every(node => 
    node.role && node.labels && node.labels.length > 0
  )
})

// Methods
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
    overlay.value = true;

    const payloadNodes = nodes.value.map(node => ({
        id: node.id,
        role: node.role,
        labels: node.labels,
    }));

    createNodesWithRoleAndLabels({ nodes: payloadNodes })
    .then(() => {
        //TODO: create a notification reused everywhere
        route.push('/dashboard');
    })
    .catch(error => {
        console.error('Error provisioning connected nodes:', error);
    })
    .finally(() => {
        overlay.value = false;
    });    
}
</script>

<style scoped>
.chip-input .v-field__input {
  padding: 0 8px !important;
}
</style>