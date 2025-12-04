<template>
    <v-row>
        <v-col v-for="provider in providers" :key="provider.key" cols="12" md="6" sm="6">
            <BaseCard class="pa-1">
                <BaseTitle :level="6" :title="provider.title" />
                <BaseRadialBarChart 
                        :key="`${provider.key}-${$i18n.locale}`"
                        :series="provider.series" 
                        :labels="translatedLabels"
                        :colors="provider.colors" />
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
    nodes: {
        type: Array,
        required: true
    }
})

// State
const rawLabels = ref(['Active', 'Provisioning', 'Pending', 'Failed']);

// Computed
const translatedLabels = computed(() =>
    rawLabels.value.map(label => t(`status.${label.toLowerCase()}`))
);
const providers = computed(() => [
    {
        key: 'onprem',
        title: t('charts.onPremises.title'),
        series: buildSeriesByProvider('onprem'),
        colors: rawLabels.value.map(label => getStatusColor(label))
    },
    {
        key: 'gcp',
        title: t('charts.gcp.title'),
        series: buildSeriesByProvider('gcp'),
        colors: rawLabels.value.map(label => getStatusColor(label))
    }
]);

// Methods
function buildSeriesByProvider(providerName) {
    return rawLabels.value.map(label =>
        props.nodes.filter(
            node =>
                node.status === label &&
                node.provider.toLowerCase() === providerName.toLowerCase()
        ).length
    );
}
</script>
