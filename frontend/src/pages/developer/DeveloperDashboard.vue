<template>
    <div>
        <v-row class="mt-2">
            <v-col cols="12" md="6" sm="6">
                <DeploymentsCharts :deployments="deploymentsChart"></DeploymentsCharts>
            </v-col>
            <v-col cols="12" md="6" sm="6">
                <BaseDataTable
                    v-model="search"
                    :headers="namespaceWithDeploymentsHeaders"
                    :items="loading ? [] : namespaceWithDeploymentsItems"
                    :loading="loading"
                    :loadingText="$t('dashboard.developer.table.loadingText')"
                    :noDataText="$t('dashboard.developer.table.noDataText')"
                    :itemsPerPageText="$t('dashboard.developer.table.itemsPerPageText')"
                    :titleToolbar="$t('dashboard.developer.table.title')"
                    :height="200"
                    :fixed-header="true"
                    rowClickable
                    @click:row="(event, item) => redirectToNamespace(item.item)"
                >
                </BaseDataTable>
            </v-col>
        </v-row>
    </div>
</template>

<script setup>
import DeploymentsCharts from '../../components/developer/DeploymentsCharts.vue'
import { ref, onMounted, computed} from 'vue'
import { listMyDeployments } from "@/services/templates.service";
import { getNamespaces } from "@/services/namespaces.service";
import { useI18n } from 'vue-i18n';
import router from '@/router';

const { t } = useI18n();

const deploymentsChart = ref([]);
const namespaces = ref([]);
const search = ref('');
const loading = ref(false);

// Computed
const namespaceWithDeploymentsHeaders = computed(() => [
    { title: t('dashboard.developer.table.headers.namespaces'), value: 'name' },
    { title: t('dashboard.developer.table.headers.numberOfDeployments'), value: 'numberOfDeployments', sortable: false, align: 'center' },
]);
const namespaceWithDeploymentsItems = computed(() => {
    if (!namespaces.value.length || !deploymentsChart.value.length) return [];

    return namespaces.value.map(ns => {
        const count = deploymentsChart.value
            .filter(d => d.namespace === ns.name).length;

        return {
            name: ns.name,
            numberOfDeployments: count
        };
    });
});

//Mounted
onMounted(async () => {
    loading.value = true;

    try {
        await Promise.all([
            fetchMyDeployments(),
            fetchNamespaces()
        ]);
    } finally {
        loading.value = false;
    }
});

// Methods
function fetchMyDeployments() {
    listMyDeployments({ template: "", namespace: "" })
        .then((response) => {            
            deploymentsChart.value = response.map(
                deployment => toCamelCaseObject(deployment)
            );            
        }).catch(error => {
             console.error("Error fetching my deployed applications:", error);
        })
}
function fetchNamespaces() {
    getNamespaces().then(response => {        
        namespaces.value = response.namespaces
            .filter(namespace => namespace.name !== "administrators");        
    }).catch(error => {
        console.error("Error fetching namespaces:", error);
    });
}
function redirectToNamespace(item) {
    router.push({ 
        name: 'administration-namespaces', 
        query: { search: item.name }
    });
}
function toCamelCaseObject(obj) {
  return Object.fromEntries(
    Object.entries(obj).map(([key, value]) => [
      key.charAt(0).toLowerCase() + key.slice(1),
      value
    ])
  );
}
</script>