<template>
    <v-row class="mt-2">
        <v-col v-for="deployment in myDeployments" :key="deployment.key" cols="12" md="6" sm="6">
            <BaseCard class="pa-1">
                <BaseTitle :level="6" :title="deployment.title" />
                <BaseRadialBarChart :key="`${deployment.key}-${$i18n.locale}`" :series="deployment.series"
                    :labels="translatedLabels" :colors="deployment.colors" />
            </BaseCard>
        </v-col>
    </v-row>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusColorHandler } from '@/composables/StatusColorHandler'

const { t } = useI18n()
const { getStatusColor } = StatusColorHandler()

const props = defineProps({
    deployments: {
        type: Array,
        required: true
    }
})

// State
const rawLabels = ref(['Healthy', 'Failed']);

// Computed
const translatedLabels = computed(() =>
    rawLabels.value.map(label => t(`status.${label.toLowerCase()}`))
);
const myDeployments = computed(() => [
    {
        key: 'deployment1',
        title: t('charts.deployments.title'),
        series: buildSeriesByDeployment(),
        colors: rawLabels.value.map(label => getStatusColor(label))
    }
]);

// Methods
function buildSeriesByDeployment() {
    return rawLabels.value.map(label =>
        props.deployments.map(deployment =>
            deployment.healthy === true ? 'Healthy' : 'Failed'
        )
        .filter(status => status === label).length
    );
}
</script>
