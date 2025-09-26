<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.title')"
            :subheading="$t('provisioning.subheading')"
            :actions="actions"
        />
        <div v-if="loading" class="mt-4 text-blue-600">{{ loadingMessage }}</div>
        <div v-if="error" class="mt-4 text-red-600">{{ errorMessage }}</div>

        <!-- Tableau des noeuds récupérés -->
        <div class="mt-4">
            <h3>{{ $t('provisioning.nodesTableTitle') }}</h3>
            <v-data-table
                :headers="nodeHeaders"
                :items="nodes"
                class="elevation-8 mt-2"
            >
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

                <template #item.labels="{ item }">
                    <div class="d-flex flex-wrap align-center">
                        <v-chip
                            v-for="(label, index) in item.labels"
                            :key="index"
                            class="ma-1"
                            elevation="2"
                            closable
                            @click:close="item.labels.splice(index, 1)"
                        >
                            {{ label }}
                        </v-chip>
                        <template v-if="!item.addingLabel">
                            <v-chip
                                class="ma-1"
                                elevation="2"
                                @click="item.addingLabel = true"
                            >
                                + Add label
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
            </v-data-table>
            <div class="d-flex justify-end">
                <BaseButton color="primary" class="mt-2" @click="validateNodes">
                    {{ $t('provisioning.validateNodesButton') }}
                </BaseButton>
            </div>
        </div>

        <!-- Tableau d'état final des noeuds -->
        <div class="mt-4">
            <h3>{{ $t('provisioning.nodeStatesTitle') }}</h3>
            <v-data-table
                :headers="stateHeaders"
                :items="nodeStates"
                class="elevation-1 mt-2"
            ></v-data-table>
            <BaseButton color="primary" class="mt-2" @click="goDashboard">
                {{ $t('provisioning.dashboardButton') }}
            </BaseButton>
        </div>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import { ref } from 'vue';


const loading = ref(false);
const loadingMessage = ref('');
const error = ref(false);
const errorMessage = ref('');
const nodes = ref([]);
const nodeStates = ref([]);

const actions = [
    {
        text: "Manage On-Premises",
        onClick: handleManageOnPremises
    },
    {
        text: "Manage cloud",
        disabled: true,
        onClick: () => alert('Export clicked!')
    }
];

const nodeHeaders = [
  { title: 'IP', value: 'IP'},
  { title: 'WID', value: 'WID'},
  { title: 'MAC', value: 'MAC', width: "20%" },
  { title: 'Rôle', value: 'role', width: "15%" },
  { title: 'Labels', value: 'labels', width: "30%" },
];

nodes.value = [
    //rajoute 25 lignes
  { IP: '192.168.0.1', WID: 'W01', MAC: 'AA:BB:CC:DD:EE:01', role: null,  labels: ["Test"] },
  { IP: '192.168.0.2', WID: 'W02', MAC: 'AA:BB:CC:DD:EE:02', role: 'Worker', labels: []},

];

const stateHeaders = [
  { title: 'Nodename', value: 'Nodename' },
  { title: 'Rôle', value: 'role' },
  { title: 'State', value: 'state' },
];

nodeStates.value = [
  { Nodename: 'node-1', role: 'Control-plane', state: 'Ready' },
  { Nodename: 'node-2', role: 'Worker', state: 'NotReady' },
];

// Methods
function handleManageOnPremises() {
    loading.value = true;
    loadingMessage.value = 'Vérification de la connexion des noeuds On-Prem...';
    error.value = false;
    nodes.value = [];
    nodeStates.value = [];

    axios.get('/api/nodes/onprem')
        .then(res => {
        loading.value = false;
        if (!res.data.length) {
            error.value = true;
            errorMessage.value = 'En attente de connexion des noeuds On-Prem…';
        } else {
            nodes.value = res.data;
        }
        })
        .catch(() => {
        loading.value = false;
        error.value = true;
        errorMessage.value = 'Erreur lors de la récupération des noeuds On-Prem';
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