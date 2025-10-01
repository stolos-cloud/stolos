<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.onPremises.title')"
            :subheading="$t('provisioning.onPremises.subheading')"
        />
        <div class="mt-4">
            <h3>{{ $t('provisioning.nodesTableTitle') }}</h3>
            <v-data-table-server
                :headers="nodeHeaders"
                :items="nodes"
                :items-length="nodes.length"
                :loading=loading
                :loading-text="$t('provisioning.onPremises.table.loadingText')"
                :no-data-text="$t('provisioning.onPremises.table.noDataText')"
                :items-per-page="10"
                class="elevation-8 mt-2"
            >
                
                <!-- Slot for roles -->
                <template #item.role="{ item }">
                    <v-select
                    v-model="item.role"
                    :items="roles"
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
                <BaseButton color="primary" class="mt-2" @click="provisionConnectedNodes">
                    {{ $t('provisioning.validateNodesButton') }}
                </BaseButton>
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
import { getConnectedNodes } from '@/services/provisioning.service';
import { onMounted, ref } from 'vue';

const roles = ['Control plane', 'Worker'];
const loading = ref(false);
const overlay = ref(false);
const nodes = ref([]);

const nodeHeaders = [
  { title: 'provisioning.onPremises.table.ip', value: 'ip'},
  { title: 'provisioning.onPremises.table.wid', value: 'wid'},
  { title: 'provisioning.onPremises.table.mac', value: 'mac', width: "20%" },
  { title: 'provisioning.onPremises.table.status', value: 'status', width: "15%" },
  { title: 'provisioning.onPremises.table.role', value: 'role', width: "15%" },
  { title: 'provisioning.onPremises.table.labels', value: 'labels', width: "30%" },
];

nodes.value = [
  { ip: '192.168.0.1', wid: 'W01', mac: 'AA:BB:CC:DD:EE:01', role: null,  labels: ["Test"] },
  { IP: '192.168.0.2', WID: 'W02', MAC: 'AA:BB:CC:DD:EE:02', role: 'Worker', labels: []},
];

//mounted
onMounted(() => {
    fetchConnectedNodes();
});


// Methods
function fetchConnectedNodes() {
    loading.value = true;
    getConnectedNodes({status: "pending"})
    .then(response => {
        nodes.value = response.data.map(node => ({
            ...node,
        }));
        console.log(nodes.value);
        
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
</script>

<style scoped>
.chip-input .v-field__input {
  padding: 0 8px !important;
}
</style>