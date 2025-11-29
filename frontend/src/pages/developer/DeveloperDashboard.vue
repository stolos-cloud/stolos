<template>
    <div>
        <DeploymentsCharts :deployments="deploymentsChart"></DeploymentsCharts>
    </div>
</template>

<script setup>
import DeploymentsCharts from '../../components/developer/DeploymentsCharts.vue'
import { ref, onMounted } from 'vue'
import { listMyDeployments } from "@/services/templates.service";

const deploymentsChart = ref([]);

//Mounted
onMounted(() => {
    fetchMyDeployments();
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
function toCamelCaseObject(obj) {
  return Object.fromEntries(
    Object.entries(obj).map(([key, value]) => [
      key.charAt(0).toLowerCase() + key.slice(1),
      value
    ])
  );
}
</script>